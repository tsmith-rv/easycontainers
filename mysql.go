package easycontainers

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"context"

	"strconv"

	"time"

	"bytes"
	"path"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// MySQL is a container using the official mysql docker image.
//
// Path is a path to a sql file, relative to the GOPATH. If set, it will run the sql in
// the file when initializing the container.
//
// Query is a string of SQL. If set, it will run the sql when initializing the container.
type MySQL struct {
	Client        *client.Client
	ContainerName string
	Port          int
	Path          string
	Query         string
}

// NewMySQL returns a new instance of MySQL and the port it will be using.
func NewMySQL(name string) (r *MySQL, port int) {
	port, err := getFreePort()
	if err != nil {
		panic(err)
	}

	c, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	return &MySQL{
		Client:        c,
		ContainerName: prefix + "mysql-" + name,
		Port:          port,
	}, port
}

// NewMySQLWithPort returns a new instance of MySQL using the specified port.
func NewMySQLWithPort(name string, port int) *MySQL {
	c, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	return &MySQL{
		Client:        c,
		ContainerName: prefix + "mysql-" + name,
		Port:          port,
	}
}

// Container spins up the mysql container and runs. When the method exits, the
// container is stopped and removed.
func (m *MySQL) Container(f func() error) error {
	CleanupContainer(m.ContainerName) // catch containers that previous cleanup missed
	defer CleanupContainer(m.ContainerName)

	ctx := context.Background()
	reader, err := m.Client.ImagePull(ctx, "docker.io/library/mysql:latest", types.ImagePullOptions{})
	if err != nil {
		return err
	}
	defer reader.Close()

	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		return err
	}

	p := nat.Port(fmt.Sprintf("%d/tcp", m.Port))

	resp, err := m.Client.ContainerCreate(
		ctx,
		&container.Config{
			Image: "mysql:latest",
			ExposedPorts: nat.PortSet{
				p: struct{}{},
			},
			Env:          []string{"MYSQL_ROOT_PASSWORD=pass"},
			AttachStdout: true,
			AttachStderr: true,
		},
		&container.HostConfig{
			PortBindings: nat.PortMap{
				p: []nat.PortBinding{
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
	defer m.Client.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{})

	err = m.Client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{})
	if err != nil {
		return err
	}

	go func() {
		r, err := m.Client.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{
			ShowStderr: true,
			ShowStdout: true,
			Follow:     true,
		})
		if err != nil {
			panic(err)
		}
		defer r.Close()

		_, err = io.Copy(os.Stdout, r)
		if err != nil {
			panic(err)
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

		// we create the table mysql.z_z_(id integer) after all the other sql has been run
		// so that we can query the table to see if all the startup sql is finished running,
		// which means that the container if fully initialized
		_, err = io.Copy(file, bytes.NewBufferString(sql+";CREATE TABLE mysql.z_z_(id integer);"))
		if err != nil {
			return err
		}

		err = m.Client.CopyToContainer(ctx, resp.ID, "/docker-entrypoint-initdb.d/schema.sql", &b, types.CopyToContainerOptions{})
		if err != nil {
			return err
		}
	}
	/*
		_, err = m.Client.ContainerExecAttach(ctx, resp.ID, types.ExecConfig{
			Cmd: []string{"until (mysql -uroot -ppass -e 'select \"initialization table found\" from mysql.z_z_ limit 1) do echo 'waiting for mysql to be up'; sleep 1; done; sleep 3;'"},
		})
		if err != nil {
			return err
		}

		fmt.Println("oiwnefoinwef")

		/*
			waitForInitializeCmd := strCmdForContainer(
				m.ContainerName,
				"until (mysql -uroot -ppass -e 'select \"initialization table found\" from mysql.z_z_ limit 1') do echo 'waiting for mysql to be up'; sleep 1; done; sleep 3;",
			)
			cmdList = append(cmdList, waitForInitializeCmd)

			for _, c := range cmdList {
				err := RunCommandWithTimeout(c, 1*time.Minute)
				if err != nil {
					// I'm showing the logs for this container specifically because if there is
					// a sql error on startup, it won't return from stderr, it will only show
					// up in the logs
					logs := Logs(m.ContainerName)
					if logs != "" {
						err = errors.New(fmt.Sprintln(err, "", " -- CONTAINER LOGS -- ", "", logs))
					}

					return err
				}
			}
	*/

	ti := time.NewTimer(5 * time.Minute)
	<-ti.C

	fmt.Println("successfully created mysql container")

	return f()
}
