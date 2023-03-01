package dynamodb

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/sirupsen/logrus"
	"github.com/telia-oss/concourse-github-lambda/pkg/repo"
)

const repoNameAttribute = "repo_name" // Must match the Terraform used to create the table

var _ repo.Lister = new(DynamoDBReposLister)

type DynamoDBReposLister struct {
	DynamoDBTableName string
	log               *logrus.Entry
	dynamoDBService   *dynamodb.DynamoDB
}

func NewDynamoDBReposLister(awsSession *session.Session, dynamoDBTableName string, logger *logrus.Logger) *DynamoDBReposLister {
	dynamoDBService := dynamodb.New(awsSession)
	log := logger.WithField("dynamodb_table", dynamoDBTableName)

	return &DynamoDBReposLister{
		DynamoDBTableName: dynamoDBTableName,
		log:               log,
		dynamoDBService:   dynamoDBService,
	}
}

func (l *DynamoDBReposLister) List() ([]*repo.Repo, error) {
	type tableItem struct {
		RepoName string `json:"repo_name"`
	}

	repos := make([]*repo.Repo, 0, 500)

	scanInput := &dynamodb.ScanInput{
		TableName: aws.String(l.DynamoDBTableName),
	}

	scanOutput, err := l.dynamoDBService.Scan(scanInput)
	if err != nil {
		l.log.Errorf("failed to scan DynamoDB table: %v", err)
		return nil, fmt.Errorf("scanning DynamoDB table: %w", err)
	}

	// TODO: Implement paging if result set exceeds 1MB in size

	for _, scanOutputItem := range scanOutput.Items {
		item := new(tableItem)
		err = dynamodbattribute.UnmarshalMap(scanOutputItem, item)
		if err != nil {
			panic(fmt.Sprintf("Failed to unmarshal Record, %v", err))
		}

		repo := &repo.Repo{
			Name:     item.RepoName,
			ReadOnly: false, // Currently all seem to be set to 'false', even for archived repos.
		}
		repos = append(repos, repo)
	}

	return repos, nil
}
