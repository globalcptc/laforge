package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gen0cide/laforge/ent"
	"github.com/gen0cide/laforge/graphql/graph"
	pb "github.com/gen0cide/laforge/grpc/proto"
	"github.com/gen0cide/laforge/grpc/server"
	"github.com/gen0cide/laforge/grpc/server/static"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const defaultPort = "80"

// Defining the Graphql handler
func graphqlHandler(client *ent.Client) gin.HandlerFunc {
	// NewExecutableSchema and Config are in the generated.go file
	// Resolver is in the resolver.go file
	h := handler.NewDefaultServer(graph.NewSchema(client))

	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

// Defining the Playground handler
func playgroundHandler() gin.HandlerFunc {
	h := playground.Handler("GraphQL", "/query")

	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

func main() {

	pgHost, ok := os.LookupEnv("PG_HOST")
	client := &ent.Client{}

	if !ok {
		client = ent.PGOpen("postgresql://laforger:laforge@127.0.0.1/laforge")
	} else {
		client = ent.PGOpen(pgHost)
	}

	ctx := context.Background()
	defer client.Close()

	// Run the auto migration tool.
	if err := client.Schema.Create(ctx); err != nil {
		log.Fatalf("failed creating schema resources: %v", err)
	}

	lis, err := net.Listen("tcp", server.Port)
	
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	router := gin.Default()

	host, ok := os.LookupEnv("HOST")

	if !ok {
		host = "localhost"
	} 

	// Add CORS middleware around every request
	// See https://github.com/rs/cors for full option listing
	router.Use(cors.New(cors.Config{
		AllowOrigins:	  []string{"http://"+host+":8080", "http://"+host+":4200", "http://"+host+":80"},
		AllowMethods:     []string{"PUT", "PATCH"},
		AllowHeaders:     []string{"Origin"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge: 12 * time.Hour,
	}))

	port, ok := os.LookupEnv("PORT")

	if !ok {
		port = defaultPort
	} 

	router.Static("/", "./dist")
	router.GET("/playground",playgroundHandler())
	router.POST("/query", graphqlHandler(client))

	go router.Run(port)


	// secure server
	certPem,certerr := static.ReadFile(server.CertFile)
	if certerr != nil {
        fmt.Println("File reading error", certerr)
        return 
	}
	keyPem,keyerr := static.ReadFile(server.KeyFile)
	if keyerr != nil {
        fmt.Println("File reading error", keyerr)
        return 
	}

	cert, tlserr := tls.X509KeyPair(certPem, keyPem)
	if tlserr != nil {
        fmt.Println("File reading error", tlserr)
        return 
	}

	creds := credentials.NewServerTLSFromCert(&cert)
	s := grpc.NewServer(grpc.Creds(creds))


	log.Printf("Starting Laforge Server on port " + server.Port)
	
	pb.RegisterLaforgeServer(s, &server.Server{
		Client: client,
		UnimplementedLaforgeServer: pb.UnimplementedLaforgeServer{},
	})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}