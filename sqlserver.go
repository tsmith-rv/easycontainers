package easycontainers

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"context"

	"strconv"

	"bytes"
	"path"

	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// SQLServer is a container using the official sqlserver docker image.
//
// Path is a path to a sql file, relative to the GOPATH. If set, it will run the sql in
// the file when initializing the container.
//
// Query is a string of SQL. If set, it will run the sql when initializing the container.
type SQLServer struct {
	Client        *client.Client
	ContainerName string
	Port          int
	Path          string
	Query         string
}

// NewSQLServer returns a new instance of SQLServer and the port it will be using.
func NewSQLServer(name string) (r *SQLServer, port int) {
	port, err := getFreePort()
	if err != nil {
		panic(err)
	}

	c, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	return &SQLServer{
		Client:        c,
		ContainerName: prefix + "sqlserver-" + name,
		Port:          port,
	}, port
}

// NewSQLServerWithPort returns a new instance of SQLServer using the specified port.
func NewSQLServerWithPort(name string, port int) *SQLServer {
	c, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	return &SQLServer{
		Client:        c,
		ContainerName: prefix + "sqlserver-" + name,
		Port:          port,
	}
}

// Container spins up the sqlserver container and runs. When the method exits, the
// container is stopped and removed.
func (m *SQLServer) Container(f func() error) error {
	ctx := context.Background()
	reader, err := m.Client.ImagePull(ctx, "mcr.microsoft.com/mssql/server:2017-latest", types.ImagePullOptions{})
	if err != nil {
		return err
	}
	defer reader.Close()

	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		return err
	}

	stopHealthCheck := make(chan struct{})

	resp, err := m.Client.ContainerCreate(
		ctx,
		&container.Config{
			Image: "mcr.microsoft.com/mssql/server:2017-latest",
			Env: []string{
				"SA_PASSWORD=Passpass_1",
				"ACCEPT_EULA=Y",
			},
			Healthcheck: &container.HealthConfig{
				Test:     []string{"CMD-SHELL", "/opt/mssql-tools/bin/sqlcmd -U SA -P Passpass_1 -b -Q 'SELECT \"startup SQL initialized\" FROM master.temp_schema.zz'"},
				Interval: 5 * time.Second,
				Timeout:  1 * time.Minute,
			},
		},
		&container.HostConfig{
			PortBindings: nat.PortMap{
				"1433/tcp": []nat.PortBinding{
					{
						HostIP:   "0.0.0.0",
						HostPort: strconv.Itoa(m.Port),
					},
				},
			},
		},
		nil,
		m.ContainerName,
	)
	if err != nil {
		return err
	}
	defer func() {
		stopHealthCheck <- struct{}{}

		m.Client.ContainerStop(ctx, resp.ID, durationPointer(30*time.Second))
		m.Client.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{
			Force: true,
		})
	}()

	err = m.Client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{})
	if err != nil {
		return err
	}

	var (
		waitUntilHealthy = make(chan struct{})
		waitUntilStarted = make(chan struct{})
	)

	go func() {
		prevState := ""
		interval := time.NewTicker(1 * time.Second)

		hasStarted := false

		for range interval.C {
			select {
			case <-stopHealthCheck:
				return
			default:
			}

			inspect, err := m.Client.ContainerInspect(ctx, resp.ID)
			if err != nil {
				panic(err)
			}

			if prevState != inspect.State.Health.Status {
				fmt.Println("STATUS CHANGE:", inspect.State.Health.Status)
				prevState = inspect.State.Health.Status

				if inspect.State.Health.Status != "starting" && !hasStarted {
					hasStarted = true
					waitUntilStarted <- struct{}{}
				}

				if inspect.State.Health.Status == "healthy" {
					for _, l := range inspect.State.Health.Log {
						fmt.Println(l.Output)
					}

					waitUntilHealthy <- struct{}{}
				}
			}
		}
	}()

	var sql string

	if m.Path != "" {
		b, err := ioutil.ReadFile(path.Join(GoPath(), m.Path))
		if err != nil {
			return err
		}

		sql = string(b)
	}

	if m.Query != "" {
		// the semicolon is in case the sql variable wasn't empty and the
		// previous sql string didn't end with a semicolon
		sql += "; " + m.Query
	}

	if sql != "" {
		file, err := ioutil.TempFile(os.TempDir(), prefix+"*.sql")
		if err != nil {
			return err
		}
		defer file.Close()
		defer os.Remove(file.Name())

		err = os.Chmod(file.Name(), 0777)
		if err != nil {
			return err
		}

		// we create the table temp_schema.zz (id integer) after all the other sql has been run
		// so that we can query the table to see if all the startup sql is finished running,
		// which means that the container if fully initialized
		_, err = io.Copy(file, bytes.NewBufferString(`
		CREATE SCHEMA temp_schema
		GO
		CREATE TABLE temp_schema.zz(id int)
		GO
		`+sql))
		if err != nil {
			return err
		}

		file.Close()

		tarContent := bytes.Buffer{}

		err = Tar(file.Name(), &tarContent)
		if err != nil {
			return err
		}

		err = m.Client.CopyToContainer(ctx, resp.ID, "/tmp", &tarContent, types.CopyToContainerOptions{})
		if err != nil {
			return err
		}

		<-waitUntilStarted

		err = dockerExec(
			ctx,
			m.Client,
			resp.ID,
			[]string{
				"/opt/mssql-tools/bin/sqlcmd",
				"-b",
				"-U",
				"SA",
				"-P",
				"Passpass_1",
				"-i",
				fmt.Sprintf("/tmp/%s", path.Base(file.Name())),
			},
		)
		if err != nil {
			return err
		}

		timeout := time.NewTimer(1 * time.Minute)

		select {
		case <-waitUntilHealthy:
			// do nothing
		case <-timeout.C:
			inspect, err := m.Client.ContainerInspect(ctx, resp.ID)
			if err != nil {
				return err
			}

			numOfLogs := len(inspect.State.Health.Log)
			lastHealthLog := ""

			if numOfLogs > 0 {
				lastHealthLog = inspect.State.Health.Log[numOfLogs-1].Output
			}

			return fmt.Errorf("timed out waiting for container to be healthy, the last healtcheck error was: %s", lastHealthLog)
		}
	}

	fmt.Println("successfully created sql server container")

	return f()
}
