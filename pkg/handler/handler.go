package handler

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/google/go-github/v29/github"
	"github.com/sirupsen/logrus"
	"github.com/telia-oss/concourse-github-lambda/pkg/manager"
	"github.com/telia-oss/concourse-github-lambda/pkg/repo"
	"github.com/telia-oss/concourse-github-lambda/pkg/team"
	"github.com/telia-oss/concourse-github-lambda/pkg/template"
)

// New lambda handler with the provided settings.
func New(manager *manager.Manager,
	githubOrganisation string,
	repoLister repo.Lister,
	tokenTemplate string,
	keyTemplate string,
	titleTemplate string,
	logger *logrus.Logger) func(team.Team) error {
	return func(team team.Team) error {
		log := logger.WithFields(logrus.Fields{
			"team": team.Name,
		})

		// Write an access token for the organisation, this is created one per team, which currently means once per lambda invocation
		tokenPath, err := template.NewTemplateWithoutRepository(team.Name, githubOrganisation, tokenTemplate).String()
		if err != nil {
			log.Warnf("failed to parse token path template: %s", err)
			return fmt.Errorf("parsing token path template: %w", err)
		}

		token, err := manager.CreateAccessToken(githubOrganisation)
		if err != nil {
			log.Warnf("failed to create access token: %s", err)
			return fmt.Errorf("creating access token: %w", err)
		}
		if err := manager.WriteSecret(tokenPath, token); err != nil {
			log.Warnf("failed to write access token: %s", err)
			return fmt.Errorf("writing access token: %w", err)
		}

		repos, err := repoLister.List()
		if err != nil {
			log.Warnf("failed to list repos: %v", err)
			return fmt.Errorf("listing repos: %w", err)
		}

	Loop:
		for _, repo := range repos {
			log := log.WithFields(logrus.Fields{
				"repository": repo.Name,
			})

			keyPath, err := template.NewTemplate(team.Name, repo.Name, githubOrganisation, keyTemplate).String()
			if err != nil {
				log.Warnf("failed to parse deploy key template: %s", err)
				continue
			}

			title, err := template.NewTemplate(team.Name, repo.Name, githubOrganisation, titleTemplate).String()
			if err != nil {
				log.Warnf("failed to parse github title template: %s", err)
				continue
			}

			// Look for existing keys belonging to the team
			keys, err := manager.ListKeys(githubOrganisation, repo.Name)
			if err != nil {
				log.Warnf("failed to list github keys: %s", err)
				continue
			}

			var oldKey *github.Key
			for _, key := range keys {
				if *key.Title == title {
					oldKey = key

					// Rotate the key if read/write permissions have changed
					if key.ReadOnly != nil && *key.ReadOnly != repo.ReadOnly {
						break
					}
					// Do not rotate if nothing has changed and the key is not >7 days old
					updated, err := manager.GetLastUpdated(keyPath)
					if err != nil {
						if e, ok := err.(awserr.Error); ok && e.Code() == secretsmanager.ErrCodeResourceNotFoundException {
							// Do not log a warning if we fail to describe because the secret does not exist.
							break
						}
						log.Warnf("failed to get last updated for secret: %s", err)
						break
					}
					if updated.After(time.Now().AddDate(0, 0, -7)) {
						continue Loop
					}
				}
			}

			// Generate a new key pair
			private, public, err := manager.GenerateKeyPair(title)
			if err != nil {
				log.Warnf("failed to generate new key pair: %s", err)
				continue
			}

			// Write the new public key to Github
			if err = manager.CreateKey(githubOrganisation, repo.Name, repo.ReadOnly, title, public); err != nil {
				log.Warnf("failed to create key on github: %s", err)
				continue
			}

			// Write the private key to Secrets manager
			if err := manager.WriteSecret(keyPath, private); err != nil {
				log.Warnf("failed to write secret key: %s", err)
				continue
			}

			// Sleep before deleting old key (in case someone has just fetched the old key)
			if oldKey != nil {
				time.Sleep(time.Second * 1)
				if err = manager.DeleteKey(githubOrganisation, repo.Name, *oldKey.ID); err != nil {
					log.Warnf("failed to delete old github key: %d: %s", *oldKey.ID, err)
					continue
				}
			}

		}
		return nil
	}
}
