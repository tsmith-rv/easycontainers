package test

import (
	"net"
	"strconv"
)

// structs for testing sql db containers
type author struct {
	ID        int    `db:"id"`
	FirstName string `db:"first_name"`
	LastName  string `db:"last_name"`
	Email     string `db:"email"`
	Birthdate string `db:"birthdate"`
	Added     string `db:"added"`
}

type post struct {
	ID          int    `db:"id"`
	AuthorID    int    `db:"author_id"`
	Title       string `db:"title"`
	Description string `db:"description"`
	Content     string `db:"content"`
	Date        string `db:"date"`
}

func isPortFree(port int) (isFree bool, err error) {
	// Concatenate a colon and the port
	host := ":" + strconv.Itoa(port)

	// Try to create a server with the port
	server, err := net.Listen("tcp", host)

	// if it fails then the port is likely taken
	if err != nil {
		return false, err
	}

	// close the server
	server.Close()

	// we successfully used and closed the port
	// so it's now available to be used again
	return true, nil
}
