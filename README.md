easycontainers
============

### WTF
I love using real services with *docker* when testing or running a quick piece of code just to see how it'll interact with an external dependency, but really hate writing scripts to spin up containers and haven't found a library that lets me abstract away all the *docker* related details in a way I like in the code. So...

### Here it is
I wrote this package. It basically wraps the necessary docker commands for spinning up / cleanup / removal of containers for common dependencies and services I work with like:

- MySQL
- RabbitMQ
- Containerized Go Apps

### Is there something that already does this?
"Probably. Don't know. Don't read. Didn't watch the movie."

### Examples

Using a MySQL container

```go
// starts the container
mysqlContainer, port := easycontainers.NewMySQL("test-container", "blog.posts")

// there is also a NewMySQLWithPort function if you want to use a specific port

// Path to a sql file to run on container startup (path is relative to GOPATH)
mysqlContainer.Path =  "/src/github.com/RedVentures/easycontainers/test/mysql-test.sql"

// runs the container and cleans up the container when the function you pass in exits
err := mysqlContainer.Container(func() error {
	// logic that needs access to the mysql container
	// container can be accessed at localhost:port using the port variable from above
})
if err != nil {
	panic(err)
}
```

Using RabbitMQ and MySQL

```go
// starts the container
mysqlContainer, mysqlPort := easycontainers.NewMySQL("test-container-mysql", "blog.posts")

rabbitContainer, rabbitPort := easycontainers.NewRabbitMQ("test-container-rabbit")

// there is also a NewMySQLWithPort function if you want to use a specific port

// Path to a sql file to run on container startup (path is relative to GOPATH)
mysqlContainer.Path =  "/src/github.com/RedVentures/easycontainers/test/mysql-test.sql"

// rabbitMQ setup
vhost := test.Vhost{  
   Name: "Import",  
}  
  
exchange := test.Exchange{  
   Name:  "data_exchange",  
   Type:  easycontainers.ExchangeTypeDirect,  
   Vhost: &vhost,  
}  
  
queue := test.Queue{  
   Name:    "ha.data_exchange.import",  
   Durable: true,  
   Vhost:   &vhost,  
}  
  
binding := test.QueueBinding{  
   Source:      exchange,  
   Destination: queue,  
   RoutingKey:  "data_import",  
   Vhost:       &vhost,  
}  
  
// when the rabbit container spins up, it will create all these resources during initialization
rabbitContainer.  
   AddVhosts(vhost).  
   AddExchanges(exchange).  
   AddQueue(queue).  
   AddBinding(binding)

// runs the containers and cleans them up when the function you pass in exits
err := rabbitContainer.Container(func() error {
	return mysqlContainer.Container(func() error {
		// logic that needs access to the mysql
		// container can be accessed at localhost:port 
		// using the mysqlPort variable from above
		
		// logic that needs access to the rabbit container
		// can be accessed at localhost:port using the rabbitPort 
		// variable from above
	})
})
if err != nil {
	panic(err)
}
```