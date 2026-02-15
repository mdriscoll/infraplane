package executor

import (
	"testing"
)

func TestValidateCommand(t *testing.T) {
	e := NewExecutor()

	tests := []struct {
		name    string
		cmd     string
		wantErr bool
		errMsg  string
	}{
		// Valid gcloud commands
		{
			name:    "gcloud list services",
			cmd:     "gcloud run services list --project=my-project --region=us-central1 --format=json",
			wantErr: false,
		},
		{
			name:    "gcloud describe instance",
			cmd:     "gcloud sql instances describe my-db --project=my-project --format=json",
			wantErr: false,
		},
		{
			name:    "gcloud secrets list",
			cmd:     "gcloud secrets list --project=my-project --format=json",
			wantErr: false,
		},
		{
			name:    "gcloud artifacts repositories list",
			cmd:     "gcloud artifacts repositories list --project=my-project --location=us-central1 --format=json",
			wantErr: false,
		},

		// Valid aws commands
		{
			name:    "aws describe instances",
			cmd:     "aws ec2 describe-instances --region us-east-1 --output json",
			wantErr: false,
		},
		{
			name:    "aws list buckets",
			cmd:     "aws s3api list-buckets --output json",
			wantErr: false,
		},
		{
			name:    "aws get function",
			cmd:     "aws lambda get-function --function-name my-func --output json",
			wantErr: false,
		},

		// Rejected: not gcloud/aws
		{
			name:    "arbitrary command",
			cmd:     "rm -rf /",
			wantErr: true,
			errMsg:  "must start with 'gcloud' or 'aws'",
		},
		{
			name:    "curl command",
			cmd:     "curl https://evil.com",
			wantErr: true,
			errMsg:  "must start with 'gcloud' or 'aws'",
		},

		// Rejected: mutation commands
		{
			name:    "gcloud deploy",
			cmd:     "gcloud run deploy my-service --image=gcr.io/my-project/my-image",
			wantErr: true,
			errMsg:  "forbidden pattern",
		},
		{
			name:    "gcloud create",
			cmd:     "gcloud sql instances create my-db --tier=db-f1-micro",
			wantErr: true,
			errMsg:  "forbidden pattern",
		},
		{
			name:    "gcloud delete",
			cmd:     "gcloud run services delete my-service",
			wantErr: true,
			errMsg:  "forbidden pattern",
		},
		{
			name:    "aws create",
			cmd:     "aws s3api create-bucket --bucket my-bucket",
			wantErr: true,
			errMsg:  "forbidden pattern",
		},
		{
			name:    "aws delete",
			cmd:     "aws s3api delete-bucket --bucket my-bucket",
			wantErr: true,
			errMsg:  "forbidden pattern",
		},
		{
			name:    "aws update",
			cmd:     "aws lambda update-function-code --function-name my-func",
			wantErr: true,
			errMsg:  "forbidden pattern",
		},

		// Rejected: shell injection
		{
			name:    "pipe injection",
			cmd:     "gcloud run services list | cat /etc/passwd",
			wantErr: true,
			errMsg:  "forbidden pattern",
		},
		{
			name:    "semicolon injection",
			cmd:     "gcloud run services list; rm -rf /",
			wantErr: true,
			errMsg:  "forbidden pattern",
		},
		{
			name:    "ampersand injection",
			cmd:     "gcloud run services list && curl evil.com",
			wantErr: true,
			errMsg:  "forbidden pattern",
		},
		{
			name:    "backtick injection",
			cmd:     "gcloud run services list `whoami`",
			wantErr: true,
			errMsg:  "forbidden pattern",
		},
		{
			name:    "subshell injection",
			cmd:     "gcloud run services list $(cat /etc/passwd)",
			wantErr: true,
			errMsg:  "forbidden pattern",
		},
		{
			name:    "redirect injection",
			cmd:     "gcloud run services list > /tmp/output",
			wantErr: true,
			errMsg:  "forbidden pattern",
		},

		// Rejected: gcloud without read action
		{
			name:    "gcloud no action",
			cmd:     "gcloud config set project my-project",
			wantErr: true,
		},

		// Rejected: aws without read action
		{
			name:    "aws no read action",
			cmd:     "aws iam put-role-policy --role-name my-role",
			wantErr: true,
		},

		// Rejected: empty
		{
			name:    "empty command",
			cmd:     "",
			wantErr: true,
			errMsg:  "empty command",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := e.ValidateCommand(tt.cmd)
			if tt.wantErr && err == nil {
				t.Errorf("expected error for command %q, got nil", tt.cmd)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for command %q: %v", tt.cmd, err)
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
