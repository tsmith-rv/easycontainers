easycontainers
==============

### WTF
I love using real services with *docker* when testing or running a quick piece of code just to see how it'll interact with an external dependency, but really hate writing scripts to spin up containers and haven't found a library that lets me abstract away all the *docker* related details in a way I like in the code. So...

### Here it is
I wrote this package. It basically hides the necessary docker commands for spinning up / cleanup / removal of containers for common dependencies and services I work with like:

- MySQL
- RabbitMQ
- Localstack
- Containerized Go Apps

The aim is to make it as simple and quick as possible to spin up these commonly used dependencies for testing or whatever without having to worry about:

- Knowing anything about *docker*
- Having knowledge of each docker image (and it's quirks)
- Cleaning up or removing old containers or images

### Is there something that already does this?
The testcontainers package (https://github.com/testcontainers/testcontainers-go) does something similar, so if that is more your game, have at it, but I
personally like the way I've done it here. Whatever works best for you.

### Prerequisites
Install *docker*. https://docs.docker.com/install/

### The Hitch
I wrote this specifically for stuff I've been testing since the Thursday before writing this, which means it isn't as thorough as it should be.
For example, I don't have helper methods for all the AWS services in Localstack, just for SQS, because that is
what I've been testing an application against. However, I would like to add to this until it covers a lot of
basic use cases for different dependencies. Which leads me to my next point...

### Contribute
Open a PR! Add support for some dependency you use - common or not, or help broaden the support for the existing
services. I have no guidelines really (which is probably bad, but whatever). I would love to get some more people
on this. It has been really helpful to a couple of things I've been working on the past few days and I can already
see myself using it a lot going forward.

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
    localstackContainer.SQS[0].SendMessage(localstackContainer.ContainerName, "some message")
 
    // send a payload to the first lambda function
    localstackContainer.Lambdas[0].SendPayload(localstackContainer.ContainerName, map[string]interface{}{
        "What is your name?": "tsmith-rv",
        "How old are you?":   108,
    })
    
    return nil
})
if err != nil {
	panic(err)
}
```