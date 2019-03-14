package test

import (
	"fmt"
	"testing"

	_ "github.com/denisenkom/go-mssqldb"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/tsmith-rv/easycontainers"
)

func Test_SqlServer_Container(t *testing.T) {
	container, port := easycontainers.NewSQLServer("Test_SqlServer_Container")

	// this tests that data is loading properly from Path and Query
	// - Path is loading the authors
	// - Query is loading the posts
	container.Path = "/src/github.com/tsmith-rv/easycontainers/test/sqlserver-test.sql"
	container.Query = `	
		CREATE TABLE blog.posts (
		  id int NOT NULL,
		  author_id int NOT NULL,
		  title varchar(255) NOT NULL,
		  description varchar(500) NOT NULL,
		  content varchar(max) NOT NULL,
		  date varchar(50) NOT NULL,
		  PRIMARY KEY (id)
		)  ;
		
		INSERT INTO blog.posts (id, author_id, title, description, content, date) VALUES (1, 1, 'Cupiditate ducimus magni error aspernatur quam eaque officia recusandae.', 'Eligendi quo harum in laboriosam voluptatum ut nemo ex. Et sapiente magni praesentium libero. Et sunt et veritatis unde quos perspiciatis amet ut.', 'Asperiores rerum harum laborum at qui quae quia. Iusto aliquam sapiente nesciunt laboriosam expedita. Eos qui delectus dolorum eligendi ipsam ad.', '1975-07-21');
		INSERT INTO blog.posts (id, author_id, title, description, content, date) VALUES (2, 2, 'Dignissimos eius voluptatem aliquid ab nostrum facere saepe voluptatem.', 'Dolorem aut et inventore rem. Ut eius eveniet qui. Error velit ea corrupti voluptas laboriosam aliquam. Blanditiis aliquam voluptas consequatur quas voluptatem.', 'Delectus qui non nesciunt ut sit omnis a. Mollitia iste ullam illum ipsam. At et voluptatibus dolores repudiandae officiis.', '1996-01-10');
		INSERT INTO blog.posts (id, author_id, title, description, content, date) VALUES (3, 3, 'Voluptas modi consequatur est id.', 'Sit culpa nemo repudiandae sint minus id. Velit eveniet aliquam tempora modi. Laboriosam molestiae ut aut omnis.', 'Qui et est recusandae qui ut in nesciunt. Maxime dolorem eligendi consectetur est dicta excepturi. Incidunt ut vel necessitatibus.', '1996-03-21');
	`

	err := container.Container(func() error {
		db, err := sqlx.Connect(
			"sqlserver",
			fmt.Sprintf(
				"sqlserver://sa:Passpass_1@localhost:%d",
				port,
			),
		)
		if err != nil {
			t.Fatal(err)
		}

		expectedAuthors := []author{
			{
				ID:        1,
				FirstName: "Terrill",
				LastName:  "Buckridge",
				Email:     "zmcglynn@example.org",
				Birthdate: "1989-03-30",
				Added:     "1976-06-06 21:51:47",
			},
			{
				ID:        2,
				FirstName: "Jamar",
				LastName:  "Buckridge",
				Email:     "lebsack.noemie@example.net",
				Birthdate: "2016-04-25",
				Added:     "2017-06-11 04:40:50",
			},
			{
				ID:        3,
				FirstName: "Alivia",
				LastName:  "McLaughlin",
				Email:     "landen.weber@example.com",
				Birthdate: "2010-01-21",
				Added:     "1980-01-31 06:20:19",
			},
			{
				ID:        4,
				FirstName: "Kristina",
				LastName:  "Schowalter",
				Email:     "yhintz@example.com",
				Birthdate: "2005-12-25",
				Added:     "2010-12-14 11:03:54",
			},
			{
				ID:        5,
				FirstName: "Norris",
				LastName:  "Gleichner",
				Email:     "derrick95@example.org",
				Birthdate: "1978-07-31",
				Added:     "2015-08-17 07:13:13",
			},
		}

		expectedPosts := []post{
			{
				ID:          1,
				AuthorID:    1,
				Title:       "Cupiditate ducimus magni error aspernatur quam eaque officia recusandae.",
				Description: "Eligendi quo harum in laboriosam voluptatum ut nemo ex. Et sapiente magni praesentium libero. Et sunt et veritatis unde quos perspiciatis amet ut.",
				Content:     "Asperiores rerum harum laborum at qui quae quia. Iusto aliquam sapiente nesciunt laboriosam expedita. Eos qui delectus dolorum eligendi ipsam ad.",
				Date:        "1975-07-21",
			},
			{
				ID:          2,
				AuthorID:    2,
				Title:       "Dignissimos eius voluptatem aliquid ab nostrum facere saepe voluptatem.",
				Description: "Dolorem aut et inventore rem. Ut eius eveniet qui. Error velit ea corrupti voluptas laboriosam aliquam. Blanditiis aliquam voluptas consequatur quas voluptatem.",
				Content:     "Delectus qui non nesciunt ut sit omnis a. Mollitia iste ullam illum ipsam. At et voluptatibus dolores repudiandae officiis.",
				Date:        "1996-01-10",
			},
			{
				ID:          3,
				AuthorID:    3,
				Title:       "Voluptas modi consequatur est id.",
				Description: "Sit culpa nemo repudiandae sint minus id. Velit eveniet aliquam tempora modi. Laboriosam molestiae ut aut omnis.",
				Content:     "Qui et est recusandae qui ut in nesciunt. Maxime dolorem eligendi consectetur est dicta excepturi. Incidunt ut vel necessitatibus.",
				Date:        "1996-03-21",
			},
		}

		actualAuthors := make([]author, 0)
		err = db.Select(&actualAuthors, "SELECT * FROM blog.authors")
		if err != nil {
			t.Fatal(err)
		}

		actualPosts := make([]post, 0)
		err = db.Select(&actualPosts, "SELECT * FROM blog.posts")
		if err != nil {
			t.Fatal(err)
		}

		if !assert.Equal(t, expectedAuthors, actualAuthors) {
			return nil
		}

		if !assert.Equal(t, expectedPosts, actualPosts) {
			return nil
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	isFree, err := isPortFree(port)
	if !assert.NoError(t, err) {
		return
	}

	if !assert.True(t, isFree, "port %d should now be available, but isn't", port) {
		return
	}
}
