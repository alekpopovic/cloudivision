package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

type Client interface {
	Clone(ctx context.Context, url, destination string) error
	Checkout(ctx context.Context, repositoryDir, ref string) error
}

type ExecClient struct{}

func (ExecClient) Clone(ctx context.Context, url, destination string) error {
	cmd := exec.CommandContext(ctx, "git", "clone", "--no-tags", url, destination)
	return runGit(cmd, "clone repository")
}

func (ExecClient) Checkout(ctx context.Context, repositoryDir, ref string) error {
	if ref == "" {
		return nil
	}
	cmd := exec.CommandContext(ctx, "git", "-C", repositoryDir, "checkout", "--detach", ref)
	return runGit(cmd, "checkout revision")
}

func runGit(cmd *exec.Cmd, action string) error {
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %s: %w", action, output.String(), err)
	}
	return nil
}
