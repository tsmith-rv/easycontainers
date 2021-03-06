easycontainers
==============

### WTF
I love using real services with *docker* when testing or running a quick piece of code just to see how it'll interact with an external dependency, but really hate writing scripts to spin up containers and haven't found a library that lets me abstract away all the *docker* related details in a way I like in the code. So...

### Here it is
I wrote this package. It basically hides the necessary docker commands for spinning up / cleanup / removal of containers for common dependencies and services I work with like:

- MySQL
- RabbitMQ
- Postgres
- Localstack
- Containerized Go Apps

The aim is to make it as simple and quick as possible to spin up these commonly used dependencies for testing or whatever without having to worry about:

- Knowing anything about *docker*
- Having knowledge of each docker image (and it's quirks)
- Cleaning up or removing old containers or images

### Is there something that already does this?
There are a couple of packages that do something similar:

- testcontainers (https://github.com/testcontainers/testcontainers-go)
- dockertest (https://github.com/ory/dockertest)

If those are more your game, have at it! I haven't used testcontainers, but dockertest is awesome, it just doesn't abstract away the docker details as much
as I would like, hence building this.

### Who isn't this for?
Anyone who needs control over the docker details and commands or needs a ton of granular control over how each service
is spun up in the container. The best use case for this is needing a very out-of-the-box configuration for these
services. As time goes on, I would love to add more granular options on all of these, but at the moment, that isn't the
case. 

The goal is to keep the docker stuff abstracted out, and to maintain simplicity in how these services are created.

### Prerequisites
Install *docker*. https://docs.docker.com/install/

### Examples

Using a MySQL container

```go
mysqlContainer, port := easycontainers.NewMySQL("test-container")
 
// there is also a NewMySQLWithPort function if you want to use a specific port
 
// Path to a sql file to run on container startup (path is relative to GOPATH)
mysqlContainer.Path =  "/src/github.com/tsmith-rv/easycontainers/test/mysql-test.sql"
 
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
mysqlContainer, mysqlPort := easycontainers.NewMySQL("test-container-mysql")
rabbitContainer, rabbitPort := easycontainers.NewRabbitMQ("test-container-rabbit")
 
// there is also a NewMySQLWithPort function if you want to use a specific port
 
// Path to a sql file to run on container startup (path is relative to GOPATH)
mysqlContainer.Path =  "/src/github.com/tsmith-rv/easycontainers/test/mysql-test.sql"
 
// Query is just a string of sql to be run on startup as well. 
mysqlContainer.Query = "CREATE DATABASE somedatabase;"
 
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

Using Localstack with Lambda functions and SQS Queues:
```go
// choose which services to spin up
localstackContainer, ports := easycontainers.NewLocalstack(
    "Test_Localstack_SQS_SendMessage",
    easycontainers.ServiceSQS,
    easycontainers.ServiceLambda,
)
 
localstackContainer.
    AddSQSQueue("queue1").
    AddSQSQueue("queue2").
    AddSQSQueue("queue3")
    
// woot is the handler name and src/github.com/tsmith-rv/easycontainers/test/handler.zip is the
// path to the zip folder relative to the GOPATH
localstackContainer.
    AddLambda("function1", "woot", "src/github.com/tsmith-rv/easycontainers/test/handler.zip").
 
err := localstackContainer.Container(func() error {
    // send a message to the first SQS queue
    localstackContainer.Queues[0].SendMessage(localstackContainer.ContainerName, "some message")
 
    // send a payload to the first lambda function
    localstackContainer.Functions[0].SendPayload(localstackContainer.ContainerName, map[string]interface{}{
        "What is your name?": "tsmith-rv",
        "How old are you?":   108,
    })
    
    return nil
})
if err != nil {
	panic(err)
}
```

### Supported OS
I'm using **macOS**, so that is all I'm positive about. Linux should be probably be fine -- Windows will not work by default. If you're using the
Linux subsystem, then maybe? I'm not sure though.

### Gotchas
- Despite going out of my way to make sure bound ports aren't given out, when running parallel tests I occasionally still
get an error from one or more docker containers claiming I'm using a port that is already allocated, despite checking if
ports are free before allocating them and also avoiding allocating the same port twice. So, a mystery for now?
- I've tried to use the smallest images possible, while still trying to use the latest versions of these services as possible,
which is a complicated balance.
- **testing** is lacking right now. There are some very basic tests that essentially make sure the container spins up and frees 
up the port when it is done.

### Things I want to add
- The ability to choose the image *tag*. While I don't want to add *docker* details into the usage, I know that being able to choose 
the version of the service you're using is essential, and image tags are the best way to implement that.
- Better tests

### Contribute
Open a PR! Add support for some dependency you use - common or not, or help broaden the support for the existing
services. I have no guidelines really (which is probably bad, but whatever). I would love to get some more people
on this. It has been really helpful to a couple of things I've been working on the past few days and I can already
see myself using it a lot going forward.