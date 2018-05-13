// Vax is a Golang AWS credentials provider using the Hashicorp Vault AWS secret
// engine.
package vax

import (
	"github.com/aws/aws-sdk-go/aws/credentials"
	vault "github.com/hashicorp/vault/api"
	"time"
)

// The VaultProvider object implements the AWS SDK `credentials.Provider`
// interface. Use the `NewVaultProvider` function to construct the object with
// default settings, or if you need to configure the `vault.Client` object,
// TTL, or path yourself, you can build the object by hand.
type VaultProvider struct {
	// The full Vault API path to the STS credentials endpoint.
	StsCredsPath string

	// The TTL of the STS credentials in the form of a Go duration string.
	TTL string

	// The `vault.Client` object used to interact with Vault.
	VaultClient *vault.Client

	expiresAt time.Time
}

// Creates a new VaultProvider. Supply the path where the AWS secrets engine
// is mounted as well as the role name to fetch from. The VaultProvider is
// initialized with a default client, which uses the VAULT_ADDR and VAULT_TOKEN
// environment variables to configure itself. This also sets a default TTL of
// 30 minutes for the credentials' lifetime.
func NewVaultProvider(enginePath string, roleName string) *VaultProvider {
	client, _ := vault.NewClient(nil)
	return &VaultProvider{
		StsCredsPath: (enginePath + "/sts/" + roleName),
		TTL:          "30m",
		VaultClient:  client,
		expiresAt:    time.Now(),
	}
}

// Implements the Retrieve() function for the AWS SDK credentials.Provider
// interface.
func (vp *VaultProvider) Retrieve() (credentials.Value, error) {
	rv := credentials.Value{
		ProviderName: "Vax",
	}

	args := make(map[string]interface{})
	args["ttl"] = vp.TTL

	resp, err := vp.VaultClient.Logical().Write(vp.StsCredsPath, args)
	if err != nil {
		return rv, err
	}

	vp.expiresAt = time.Now().Add(time.Duration(resp.LeaseDuration) * time.Second)

	rv.AccessKeyID = resp.Data["access_key"].(string)
	rv.SecretAccessKey = resp.Data["secret_key"].(string)
	rv.SessionToken = resp.Data["security_token"].(string)

	return rv, nil
}

// Implements the IsExpired() function for the AWS SDK credentials.Provider
// interface.
func (vp *VaultProvider) IsExpired() bool {
	// report expiration 10 seconds before the actual expiration time
	return time.Now().After(vp.expiresAt.Add(-10 * time.Second))
}
