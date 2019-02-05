package test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/RedVentures/easycontainers"
)

func Test_MySQL_Container(t *testing.T) {
	container, port := easycontainers.NewMySQL("test-container", "blog.posts")
	if port < 5000 || port > 6000 {
		t.Error(errors.New("port should be with the range of 5000-6000"))
		return
	}

	container.Path = "/src/github.com/RedVentures/easycontainers/test/mysql-test.sql"

	err := container.Container(func() error {
		fmt.Println("port is", port)
		fmt.Println("container is up")

		return nil
	})
	if err != nil {
		t.Error(err)
		return
	}
}
