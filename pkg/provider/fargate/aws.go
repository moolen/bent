package fargate

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
)

func newSession() (*session.Session, error) {
	return session.NewSession()
}

func newSessionWithRole(role string) (*session.Session, error) {
	defaultSession, err := newSession()
	if err != nil {
		return nil, err
	}
	svc := sts.New(defaultSession)
	result, err := svc.AssumeRole(&sts.AssumeRoleInput{
		DurationSeconds: aws.Int64(900),
		RoleArn:         aws.String(role),
		RoleSessionName: aws.String("bent"),
	})
	if err != nil {
		return nil, err
	}
	return session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentialsFromCreds(credentials.Value{
			AccessKeyID:     *result.Credentials.AccessKeyId,
			SecretAccessKey: *result.Credentials.SecretAccessKey,
			SessionToken:    *result.Credentials.SessionToken,
		}),
	})
}
