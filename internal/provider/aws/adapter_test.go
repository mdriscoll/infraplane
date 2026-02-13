package aws

import (
	"context"
	"testing"

	"github.com/matthewdriscoll/infraplane/internal/domain"
)

func TestAdapter_Provider(t *testing.T) {
	a := NewAdapter(&Config{Region: "us-east-1"})
	if a.Provider() != domain.ProviderAWS {
		t.Errorf("provider = %v, want aws", a.Provider())
	}
}

func TestAdapter_ValidateCredentials(t *testing.T) {
	ctx := context.Background()

	t.Run("missing credentials", func(t *testing.T) {
		a := NewAdapter(&Config{Region: "us-east-1"})
		err := a.ValidateCredentials(ctx)
		if err == nil {
			t.Error("expected error for missing credentials")
		}
	})

	t.Run("valid credentials", func(t *testing.T) {
		a := NewAdapter(&Config{
			Region:          "us-east-1",
			AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
			SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		})
		err := a.ValidateCredentials(ctx)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestAdapter_ApplyTerraform(t *testing.T) {
	ctx := context.Background()
	a := NewAdapter(&Config{
		Region:          "us-west-2",
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
	})

	t.Run("successful apply", func(t *testing.T) {
		result, err := a.ApplyTerraform(ctx, `resource "aws_db_instance" "db" {}`)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if result == "" {
			t.Error("expected non-empty plan output")
		}
	})

	t.Run("empty HCL", func(t *testing.T) {
		_, err := a.ApplyTerraform(ctx, "")
		if err == nil {
			t.Error("expected error for empty HCL")
		}
	})

	t.Run("no credentials", func(t *testing.T) {
		noCreds := NewAdapter(&Config{Region: "us-east-1"})
		_, err := noCreds.ApplyTerraform(ctx, `resource "aws_db_instance" "db" {}`)
		if err == nil {
			t.Error("expected error for missing credentials")
		}
	})
}

func TestAdapter_DestroyTerraform(t *testing.T) {
	ctx := context.Background()
	a := NewAdapter(&Config{
		Region:          "us-east-1",
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
	})

	t.Run("successful destroy", func(t *testing.T) {
		err := a.DestroyTerraform(ctx, `resource "aws_db_instance" "db" {}`)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("empty HCL", func(t *testing.T) {
		err := a.DestroyTerraform(ctx, "")
		if err == nil {
			t.Error("expected error for empty HCL")
		}
	})
}
