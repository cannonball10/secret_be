package secret

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	smtypes "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/cannonball10/foundation/errors/secret"
)

// SecretManagerConnector implements SecretConnector using AWS Secrets Manager (AWS SDK v2).
// It can be constructed with an eager client or lazily via a factory.
type SecretManagerConnector struct {
	client        *secretsmanager.Client
	clientOnce    sync.Once
	clientFactory func() *secretsmanager.Client
}

// NewSecretManagerConnector returns a connector with an eagerly provided client.
func NewSecretManagerConnector(client *secretsmanager.Client) *SecretManagerConnector {
	return newSecretManagerConnector(client, nil)
}

// NewSecretManagerConnectorLazy returns a connector that initializes the client on first use.
func NewSecretManagerConnectorLazy(factory func() *secretsmanager.Client) *SecretManagerConnector {
	return newSecretManagerConnector(nil, factory)
}

func DefaultSecretManagerConnector(ctx context.Context) (*SecretManagerConnector, error) {
	awsCfg, err := awscfg.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("aws config LoadDefaultConfig: %w", err)
	}
	return NewSecretManagerConnector(secretsmanager.NewFromConfig(awsCfg)), nil
}

// NewSecretManagerConnectorFromDefaultConfig loads AWS default config and creates a connector.
func NewSecretManagerConnectorFromDefaultConfig(ctx context.Context) (*SecretManagerConnector, error) {
	awsCfg, err := awscfg.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("aws config LoadDefaultConfig: %w", err)
	}
	return NewSecretManagerConnector(secretsmanager.NewFromConfig(awsCfg)), nil
}

func newSecretManagerConnector(client *secretsmanager.Client, factory func() *secretsmanager.Client) *SecretManagerConnector {
	return &SecretManagerConnector{
		client:        client,
		clientFactory: factory,
	}
}

func (c *SecretManagerConnector) Get(ctx context.Context, key string) (any, error) {
	if key == "" {
		return nil, secret.ErrSecretKeyEmpty
	}
	client, err := c.getClient()
	if err != nil {
		return nil, err
	}

	in := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(key),
	}

	out, err := client.GetSecretValue(ctx, in)
	if err != nil {
		slog.ErrorContext(ctx, "secretsmanager GetSecretValue failed", "secret_name", key, "error", err)
		return nil, fmt.Errorf("secretsmanager GetSecretValue %q: %w", key, err)
	}

	if out.SecretString != nil {
		s := aws.ToString(out.SecretString)
		// Best-effort JSON decoding: if it's valid JSON, return decoded value;
		// otherwise return the raw string.
		var decoded any
		if json.Unmarshal([]byte(s), &decoded) == nil {
			return decoded, nil
		}
		return s, nil
	}

	if out.SecretBinary != nil {
		// Copy to ensure callers can't mutate underlying buffer.
		b := make([]byte, len(out.SecretBinary))
		copy(b, out.SecretBinary)
		return b, nil
	}

	return nil, nil
}

func (c *SecretManagerConnector) Set(ctx context.Context, key string, value any) error {
	if key == "" {
		return secret.ErrSecretKeyEmpty
	}
	client, err := c.getClient()
	if err != nil {
		return err
	}

	secretString, secretBinary, err := encodeSecretValue(value)
	if err != nil {
		return err
	}

	createIn := &secretsmanager.CreateSecretInput{
		Name: aws.String(key),
	}
	if secretString != nil {
		createIn.SecretString = secretString
	}
	if secretBinary != nil {
		createIn.SecretBinary = secretBinary
	}

	_, err = client.CreateSecret(ctx, createIn)
	if err == nil {
		return nil
	}

	var exists *smtypes.ResourceExistsException
	if errors.As(err, &exists) {
		// If it already exists, treat Set as "upsert" via a new version.
		return c.putSecretValue(ctx, client, key, secretString, secretBinary)
	}

	slog.ErrorContext(ctx, "secretsmanager CreateSecret failed", "secret_name", key, "error", err)
	return fmt.Errorf("secretsmanager CreateSecret %q: %w", key, err)
}

func (c *SecretManagerConnector) Update(ctx context.Context, key string, value any) error {
	if key == "" {
		return secret.ErrSecretKeyEmpty
	}
	client, err := c.getClient()
	if err != nil {
		return err
	}

	secretString, secretBinary, err := encodeSecretValue(value)
	if err != nil {
		return err
	}

	if err := c.putSecretValue(ctx, client, key, secretString, secretBinary); err != nil {
		slog.ErrorContext(ctx, "secretsmanager Update failed", "secret_name", key, "error", err)
		return err
	}
	return nil
}

func (c *SecretManagerConnector) Delete(ctx context.Context, key string) error {
	if key == "" {
		return secret.ErrSecretKeyEmpty
	}
	client, err := c.getClient()
	if err != nil {
		return err
	}

	in := &secretsmanager.DeleteSecretInput{
		SecretId: aws.String(key),
	}

	_, err = client.DeleteSecret(ctx, in)
	if err != nil {
		slog.ErrorContext(ctx, "secretsmanager DeleteSecret failed", "secret_name", key, "error", err)
		return fmt.Errorf("secretsmanager DeleteSecret %q: %w", key, err)
	}
	return nil
}

func (c *SecretManagerConnector) getClient() (*secretsmanager.Client, error) {
	c.clientOnce.Do(func() {
		if c.client == nil && c.clientFactory != nil {
			c.client = c.clientFactory()
		}
	})
	if c.client == nil {
		return nil, secret.ErrSecretClientUninitialized
	}
	return c.client, nil
}

// Close releases any resources held by the Secrets Manager client.
func (c *SecretManagerConnector) Close() error {
	// AWS SDK v2 Secrets Manager client doesn't have an explicit Close method,
	// but we nil out the reference to allow garbage collection.
	c.client = nil
	return nil
}

func (c *SecretManagerConnector) putSecretValue(ctx context.Context, client *secretsmanager.Client, key string, secretString *string, secretBinary []byte) error {
	in := &secretsmanager.PutSecretValueInput{
		SecretId: aws.String(key),
	}
	if secretString != nil {
		in.SecretString = secretString
	}
	if secretBinary != nil {
		in.SecretBinary = secretBinary
	}
	_, err := client.PutSecretValue(ctx, in)
	if err != nil {
		return fmt.Errorf("secretsmanager PutSecretValue %q: %w", key, err)
	}
	return nil
}

func encodeSecretValue(v any) (*string, []byte, error) {
	switch x := v.(type) {
	case nil:
		// Store explicit JSON null (rather than empty string).
		s := "null"
		return &s, nil, nil
	case string:
		return aws.String(x), nil, nil
	case []byte:
		return nil, x, nil
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return nil, nil, fmt.Errorf("secret json marshal: %w", err)
		}
		s := string(b)
		return &s, nil, nil
	}
}
