package easycontainers

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"time"
)

// Postgres is a container using the official postgres docker image.
//
// Path is a path to a sql file, relative to the GOPATH. If set, it will run the sql in
// the file when initializing the container.
//
// Query is a string of SQL. If set, it will run the sql when initializing the container.
type Postgres struct {
	ContainerName string
	Port          int
	Path          string
	Query         string
}

// NewPostgres returns a new instance of Postgres and the port it will be using.
func NewPostgres(name string) (r *Postgres, port int) {
	port, err := getFreePort()
	if err != nil {
		panic(err)
	}

	return &Postgres{
		ContainerName: prefix + "postgres-" + name,
		Port:          port,
	}, port
}

// NewPostgresWithPort returns a new instance of Postgres using the specified port.
func NewPostgresWithPort(name string, port int) *Postgres {
	return &Postgres{
		ContainerName: prefix + "postgres-" + name,
		Port:          port,
	}
}

// Container spins up the postgres container and runs. When the method exits, the
// container is stopped and removed.
func (m *Postgres) Container(f func() error) error {
	CleanupContainer(m.ContainerName) // catch containers that previous cleanup missed
	defer CleanupContainer(m.ContainerName)

	var cmdList []*exec.Cmd

	runContainerCmd := exec.Command(
		"docker",
		"run",
		"--rm",
		"-p",
		fmt.Sprintf("%d:5432", m.Port),
		"--name",
		m.ContainerName,
		"-e",
		"POSTGRES_PASSWORD=pass",
		"-d",
		"postgres:latest",
	)
	cmdList = append(cmdList, runContainerCmd)

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

		// we create the table postgres.z_z_(id integer) after all the other sql has been run
		// so that we can query the table to see if all the startup sql is finished running,
		// which means that the container if fully initialized
		_, err = io.Copy(file, bytes.NewBufferString(sql+";CREATE TABLE postgres.public.z_z_(id INTEGER);"))
		if err != nil {
			return err
		}

		file.Close()

		addStartupSQLFileCmd := exec.Command(
			"/bin/bash",
			"-c",
			fmt.Sprintf(
				`docker cp %s $(docker ps --filter="name=^/%s$" --format="{{.ID}}"):/docker-entrypoint-initdb.d`,
				file.Name(),
				m.ContainerName,
			),
		)
		cmdList = append(cmdList, addStartupSQLFileCmd)
	}

	waitForInitializeCmd := strCmdForContainer(
		m.ContainerName,
		"until (psql -U postgres -h localhost -c \"select 1 from postgres.public.z_z_ limit 1\") do echo \"waiting for postgres to be up\"; sleep 1; done; sleep 3;",
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

	fmt.Println("successfully created postgres container")

	return f()
}
