package main

import (
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/joho/godotenv"

	"github.com/kxplxn/goteam/internal/tasksvc/taskapi"
	"github.com/kxplxn/goteam/internal/tasksvc/tasksapi"
	"github.com/kxplxn/goteam/pkg/api"
	"github.com/kxplxn/goteam/pkg/cookie"
	"github.com/kxplxn/goteam/pkg/db/tasktbl"
	"github.com/kxplxn/goteam/pkg/log"
)

const (
	// envPort is the name of the environment variable used for setting the port
	// to run the task service on.
	envPort = "TASK_SERVICE_PORT"

	// envAWSEndpoint is the name of the environment variable used for setting
	// the AWS endpoint to connect to for DynamoDB. It should only be non-empty
	// on local pointing to the local DynamoDB instance.
	envAWSEndpoint = "AWS_ENDPOINT"

	// envPort is the name of the environment variable used for providing AWS
	// access key to the DynamoDB client.
	envAWSAccessKey = "AWS_ACCESS_KEY"

	// envPort is the name of the environment variable used for providing AWS
	// secret key to the DynamoDB client.
	envAWSSecretKey = "AWS_SECRET_KEY"

	// envAWSRegion is the name of the environment variable used for determining
	// the AWS region to connect to for DynamoDB.
	envAWSRegion = "AWS_REGION"

	// envJWTKey is the name of the environment variable used for signing JWTs.
	envJWTKey = "JWT_KEY"

	// envClientOrigin is the name of the environment variable used to set up
	// CORS with the client app.
	envClientOrigin = "CLIENT_ORIGIN"
)

func main() {
	// create a logger
	log := log.New()

	// load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
		return
	}

	// get environment variables
	var (
		port         = os.Getenv(envPort)
		awsEndpoint  = os.Getenv(envAWSEndpoint)
		awsAccessKey = os.Getenv(envAWSAccessKey)
		awsSecretKey = os.Getenv(envAWSSecretKey)
		awsRegion    = os.Getenv(envAWSRegion)
		jwtKey       = os.Getenv(envJWTKey)
		clientOrigin = os.Getenv(envClientOrigin)
	)

	// check all environment variables were set
	// - except aws endpoint, which is only set on local
	errPostfix := "was empty"
	switch "" {
	case port:
		log.Fatal(envPort, errPostfix)
		return
	case awsAccessKey:
		log.Fatal(envAWSAccessKey, errPostfix)
		return
	case awsSecretKey:
		log.Fatal(envAWSSecretKey, errPostfix)
		return
	case awsRegion:
		log.Fatal(envAWSRegion, errPostfix)
		return
	case jwtKey:
		log.Fatal(envJWTKey, errPostfix)
		return
	case clientOrigin:
		log.Fatal(envClientOrigin, errPostfix)
		return
	}

	// define aws config
	cfg := aws.Config{
		Region: awsRegion,
		Credentials: credentials.NewStaticCredentialsProvider(
			awsAccessKey, awsSecretKey, "",
		),
	}
	if awsEndpoint != "" {
		cfg.BaseEndpoint = aws.String(awsEndpoint)
	}

	// create DynamoDB client from config
	db := dynamodb.NewFromConfig(cfg)

	// create auth decoder to be used by API handlers
	authDecoder := cookie.NewAuthDecoder([]byte(jwtKey))

	// register handlers for HTTP routes
	mux := http.NewServeMux()

	taskTitleValidator := taskapi.NewTitleValidator()
	mux.Handle("/task", api.NewHandler(map[string]api.MethodHandler{
		http.MethodPost: taskapi.NewPostHandler(
			authDecoder,
			taskapi.ValidatePostReq,
			tasktbl.NewInserter(db),
			log,
		),
		http.MethodPatch: taskapi.NewPatchHandler(
			authDecoder,
			taskTitleValidator,
			taskTitleValidator,
			tasktbl.NewUpdater(db),
			log,
		),
		http.MethodDelete: taskapi.NewDeleteHandler(
			authDecoder,
			tasktbl.NewDeleter(db),
			log,
		),
	}))

	mux.Handle("/tasks", api.NewHandler(map[string]api.MethodHandler{
		http.MethodPatch: tasksapi.NewPatchHandler(
			authDecoder,
			tasksapi.NewColNoValidator(),
			tasktbl.NewMultiUpdater(db),
			log,
		),
		http.MethodGet: tasksapi.NewGetHandler(
			tasksapi.NewBoardIDValidator(),
			tasktbl.NewRetrieverByBoard(db),
			authDecoder,
			tasktbl.NewRetrieverByTeam(db),
			log,
		),
	}))

	// serve the registered routes
	log.Info("running task service on port", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
		return
	}
}
