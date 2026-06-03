package sopsutil

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/getsops/sops/v3"
	"github.com/getsops/sops/v3/aes"
	sopsage "github.com/getsops/sops/v3/age"
	"github.com/getsops/sops/v3/cmd/sops/common"
	"github.com/getsops/sops/v3/config"
	"github.com/getsops/sops/v3/decrypt"
	"github.com/getsops/sops/v3/keyservice"
	"github.com/getsops/sops/v3/stores"
	"github.com/getsops/sops/v3/stores/yaml"
	"github.com/getsops/sops/v3/version"
	ghodssyaml "github.com/ghodss/yaml"
)

// IsSOPSYAML reports whether data looks like a SOPS-encrypted YAML document.
func IsSOPSYAML(data []byte) bool {
	var decoded map[string]interface{}
	if err := ghodssyaml.Unmarshal(data, &decoded); err != nil {
		return false
	}
	meta, ok := decoded[stores.SopsMetadataKey]
	if !ok || meta == nil {
		return false
	}
	m, ok := meta.(map[string]interface{})
	if !ok {
		return true
	}
	_, hasMAC := m["mac"]
	_, hasVersion := m["version"]
	return hasMAC || hasVersion
}

// DecryptYAML decrypts a SOPS YAML document using SOPS' Go API.
func DecryptYAML(path string, data []byte) ([]byte, error) {
	cleartext, err := decrypt.Data(data, "yaml")
	if err != nil {
		return nil, fmt.Errorf("decrypting %s: %w", path, err)
	}
	return cleartext, nil
}

// EncryptOptions controls YAML encryption.
type EncryptOptions struct {
	AgeRecipients    []string
	EncryptedRegex   string
	UnencryptedRegex string
}

// EncryptYAML encrypts plaintext YAML using SOPS and Age recipients. If no Age
// recipients are supplied, it attempts to load a matching .sops.yaml creation rule.
func EncryptYAML(path string, data []byte, opts EncryptOptions) ([]byte, error) {
	store := yaml.NewStore(&config.NewStoresConfig().YAML)
	branches, err := store.LoadPlainFile(data)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling YAML: %w", err)
	}
	if len(branches) == 0 {
		return nil, fmt.Errorf("file cannot be empty")
	}
	if store.HasSopsTopLevelKey(branches[0]) {
		return nil, fmt.Errorf("file already contains top-level %q metadata", stores.SopsMetadataKey)
	}

	conf, err := encryptionConfig(path, opts)
	if err != nil {
		return nil, err
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	tree := sops.Tree{
		Branches: branches,
		Metadata: sops.Metadata{
			KeyGroups:               conf.KeyGroups,
			UnencryptedSuffix:       conf.UnencryptedSuffix,
			EncryptedSuffix:         conf.EncryptedSuffix,
			UnencryptedRegex:        conf.UnencryptedRegex,
			EncryptedRegex:          conf.EncryptedRegex,
			UnencryptedCommentRegex: conf.UnencryptedCommentRegex,
			EncryptedCommentRegex:   conf.EncryptedCommentRegex,
			MACOnlyEncrypted:        conf.MACOnlyEncrypted,
			Version:                 version.Version,
			ShamirThreshold:         conf.ShamirThreshold,
			LastModified:            time.Now().UTC(),
		},
		FilePath: absPath,
	}

	dataKey, errs := tree.GenerateDataKeyWithKeyServices([]keyservice.KeyServiceClient{keyservice.NewLocalClient()})
	if len(errs) > 0 {
		return nil, fmt.Errorf("generating SOPS data key: %s", joinErrors(errs))
	}
	if err := common.EncryptTree(common.EncryptTreeOpts{Tree: &tree, Cipher: aes.NewCipher(), DataKey: dataKey}); err != nil {
		return nil, err
	}
	out, err := store.EmitEncryptedFile(tree)
	if err != nil {
		return nil, fmt.Errorf("marshalling encrypted YAML: %w", err)
	}
	return out, nil
}

func encryptionConfig(path string, opts EncryptOptions) (*config.Config, error) {
	if len(opts.AgeRecipients) > 0 {
		keys, err := sopsage.MasterKeysFromRecipients(strings.Join(opts.AgeRecipients, ","))
		if err != nil {
			return nil, err
		}
		group := make(sops.KeyGroup, 0, len(keys))
		for _, key := range keys {
			group = append(group, key)
		}
		conf := &config.Config{
			KeyGroups:         []sops.KeyGroup{group},
			UnencryptedSuffix: sops.DefaultUnencryptedSuffix,
			EncryptedRegex:    opts.EncryptedRegex,
			UnencryptedRegex:  opts.UnencryptedRegex,
		}
		if opts.EncryptedRegex != "" || opts.UnencryptedRegex != "" {
			conf.UnencryptedSuffix = ""
		}
		return conf, nil
	}

	confPath, err := config.FindConfigFile(path)
	if err != nil {
		return nil, fmt.Errorf("no --age recipients supplied and no .sops.yaml creation rule found: %w", err)
	}
	conf, err := config.LoadCreationRuleForFile(confPath, path, nil)
	if err != nil {
		return nil, err
	}
	if opts.EncryptedRegex != "" {
		conf.EncryptedRegex = opts.EncryptedRegex
		conf.UnencryptedSuffix = ""
	}
	if opts.UnencryptedRegex != "" {
		conf.UnencryptedRegex = opts.UnencryptedRegex
		conf.UnencryptedSuffix = ""
	}
	return conf, nil
}

func joinErrors(errs []error) string {
	parts := make([]string, 0, len(errs))
	for _, err := range errs {
		if err != nil {
			parts = append(parts, err.Error())
		}
	}
	return strings.Join(parts, "; ")
}

func DecryptYAMLFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return DecryptYAML(path, data)
}

func LooksLikeEncryptedScalar(data []byte) bool {
	return bytes.Contains(data, []byte("ENC[AES256_GCM"))
}
