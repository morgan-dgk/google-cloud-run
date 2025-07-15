package main

import (
	run "cloud.google.com/go/run/apiv2"
	runpb "cloud.google.com/go/run/apiv2/runpb"

	"context"
	"fmt"
	"time"

	"google.golang.org/api/option"

	"dagger/google-cloud-run/internal/dagger"

	"google.golang.org/protobuf/types/known/durationpb"
)

type CloudRunJob struct {
	Name           string
	Project        string
	Location       string
	ServiceAccount string
	Image          string
	MaxRetries     int32
	Timeout        string
	Args           []string
}

func (m *GoogleCloudRun) Job(name string, project string, location string, image string, service_account string) *CloudRunJob {
	return &CloudRunJob{
		Name:           name,
		Project:        project,
		Location:       location,
		Image:          image,
		ServiceAccount: service_account,
		Timeout:        "600s",
		MaxRetries:     3,
	}
}

func (m *CloudRunJob) WithMaxRetries(max_retries int32) *CloudRunJob {
	m.MaxRetries = max_retries
	return m
}

func (m *CloudRunJob) WithTimeout(timeout string) *CloudRunJob {
	m.Timeout = timeout
	return m
}

func (m *CloudRunJob) WithArgs(args []string) *CloudRunJob {
	m.Args = args
	return m
}

// TODO
func (m *CloudRunJob) WithVolumes() {}

func (m *CloudRunJob) Create(ctx context.Context, credential *dagger.Secret) (string, error) {
	json, err := credential.Plaintext(ctx)
	b := []byte(json)
	gcrClient, err := run.NewJobsClient(ctx, option.WithCredentialsJSON(b))

	if err != nil {
		return "", fmt.Errorf("Failed to create Google Cloud Run client: %w", err)
	}

	parsed_timeout, err := time.ParseDuration(m.Timeout)

	if err != nil {
    return "", fmt.Errorf("Failed to parse timeout `%s`: %w", m.Timeout, err)
	}

	defer gcrClient.Close()

	gcrJobRequest := &runpb.CreateJobRequest{
		Parent: fmt.Sprintf("projects/%s/locations/%s", m.Project, m.Location),
		JobId:  m.Name,
		Job: &runpb.Job{
			Template: &runpb.ExecutionTemplate{
				Template: &runpb.TaskTemplate{
					Containers: []*runpb.Container{
						{
							Image: m.Image,
							Args:  m.Args,
							// Env:   []*runpb.EnvVar,
						},
					},
					ServiceAccount: m.ServiceAccount,
					Timeout:        durationpb.New(parsed_timeout),
					Retries: &runpb.TaskTemplate_MaxRetries{
						MaxRetries: m.MaxRetries,
					},
				},
			},
		},
	}

	gcrOperation, err := gcrClient.CreateJob(ctx, gcrJobRequest)

	if err != nil {
    return "", fmt.Errorf("Create Cloud Run Job request failed: %w", err)
	}

	gcrResponse, err := gcrOperation.Wait(ctx)

	if err != nil {
    return "", fmt.Errorf("Failed to create Cloud Run Job: %w", err)
	}

	return gcrResponse.GetName(), nil
}
