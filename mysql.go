package easycontainers

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
)

// MySQL is a container using the official mysql docker image.
//
// Path is a path to a sql file, relative to the GOPATH. If set, it will run the sql in
// the file when initializing the container.
//
// Query is a string of SQL. If set, it will run the sql when initializing the container.
//
// InitializeTable is the table to query to make sure the initialization is finished,
// because once mysql starts in the container, the queries from Path and Query may
// still not be finished running. If neither Path nor Query is set, you can set it to
// information_schema.COLUMNS
type MySQL struct {
	ContainerName   string
	Port            int
	Path            string
	Query           string
	InitializeTable string
}

// NewMySQL returns a new instance of MySQL and the port it will be using, which is
// a randomly selected number between 5000-6000.
//
// Conflicts are possible because it doesn't check if the port is already allocated.
func NewMySQL(name string, initialTable string) (r *MySQL, port int) {
	port = 5000 + rand.Intn(1000)

	return &MySQL{
		ContainerName:   "mysql-" + name,
		Port:            port,
		InitializeTable: initialTable,
	}, port
}

// NewMySQLWithPort returns a new instance of MySQL using the specified port.
func NewMySQLWithPort(name string, initialTable string, port int) *MySQL {
	return &MySQL{
		ContainerName:   "mysql-" + name,
		Port:            port,
		InitializeTable: initialTable,
	}
}

// Container spins up the mysql container and runs. When the method exits, the
// container is stopped and removed.
func (m *MySQL) Container(f func() error) error {
	CleanupContainer(m.ContainerName) // catch containers that previous cleanup missed
	defer CleanupContainer(m.ContainerName)

	var cmdList []*exec.Cmd

	runContainerCmd := exec.Command(
		"docker",
		"run",
		"--rm",
		"-p",
		fmt.Sprintf("%d:3306", m.Port),
		"--name",
		m.ContainerName,
		"-e",
		"MYSQL_ROOT_PASSWORD=pass",
		"-d",
		"mysql:latest",
	)

	cmdList = append(cmdList, runContainerCmd)

	if m.Path != "" {
		if !strings.HasSuffix(m.Path, ".sh") &&
			!strings.HasSuffix(m.Path, ".sql") &&
			!strings.HasSuffix(m.Path, ".sql.gz") {
			return errors.New(
				"file specified by Path should have an extension of .sh, .sql, or .sql.gz or it won't be run during initialization",
			)
		}

		cmdList = append(cmdList, exec.Command(
			"/bin/bash",
			"-c",
			fmt.Sprintf(
				`docker cp %s $(docker ps --filter="name=%s" --format="{{.ID}}"):/docker-entrypoint-initdb.d`,
				path.Join(GoPath(), m.Path),
				m.ContainerName,
			),
		))
	}

	if m.Query != "" {
		fileName := strconv.Itoa(rand.Intn(100000)*2) + ".sql"

		file, err := os.Create(fileName)
		if err != nil {
			return err
		}

		_, err = io.Copy(file, bytes.NewBufferString(m.Query))
		if err != nil {
			return err
		}

		err = file.Close()
		if err != nil {
			return err
		}

		defer func() {
			os.Remove(fileName)
		}()

		cmdList = append(cmdList, exec.Command(
			"/bin/bash",
			"-c",
			fmt.Sprintf(
				`docker cp %s $(docker ps --filter="name=%s" --format="{{.ID}}"):/docker-entrypoint-initdb.d`,
				fileName,
				m.ContainerName,
			),
		))
	}

	waitForInitializeCmd := strCmdForContainer(
		m.ContainerName,
		fmt.Sprintf(
			"until (mysql -uroot -ppass -e 'select 1 from %s limit 1') do echo 'waiting for mysql to be up'; sleep 1; done; sleep 3;",
			m.InitializeTable,
		),
	)
	cmdList = append(cmdList, waitForInitializeCmd)

	for _, c := range cmdList {
		err := RunCommandWithTimeout(c)
		if err != nil {
			return err
		}
	}

	fmt.Println("successfully created mysql container")

	return f()
}
