package dynamodb

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/sirupsen/logrus"
)

func TestDynamoDBIntegrationGetsAllItemsInTable(t *testing.T) {
	const dynamodbTableName = "ew1-shared-github-repos-db"

	awsSession, err := session.NewSession()
	if err != nil {
		t.Fatalf("failed to create AWS session: %v", err)
	}

	lister := NewDynamoDBReposLister(awsSession, dynamodbTableName, logrus.StandardLogger())

	repos, err := lister.List()
	if err != nil {
		t.Fatalf("expected no error listing repos, got %v (of type %T)", err, err)
	}
	if len(repos) == 0 {
		t.Fatal("expected items in list of repos, but was none")
	}

	t.Logf("listed %d repos", len(repos))
	
	for _, repo := range repos { 
		t.Logf("got repo: %v", repo)
	}
}
