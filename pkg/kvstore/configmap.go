package kvstore

// ConfigMapKvStore is a KvStore implementation backed by Kubernetes
// ConfigMaps, one ConfigMap per key.
//
// Every key is stored in its own dedicated ConfigMap named
// "kvstore-<storeName>-<keyName>" that holds exactly one data entry (the key
// name as the data key). Using one ConfigMap per key instead of a single
// shared ConfigMap removes the noisy-neighbor problem: a value is bounded
// only by the 1 MiB data limit of its own ConfigMap, not by every other value
// already stored by the same store. A large value therefore can not push the
// store past its limit and make unrelated keys unwritable, and writing one
// large key does not rewrite (and conflict with) the whole object.
//
// This implementation is not meant to be used for high-performance caches,
// but for use-cases that can accept relaxed response times (e.g. metadata
// refreshes). Every read is a ConfigMap read, every write is an optimistic
// create-or-update of a single object, and listing (EstimateSize) is a
// ConfigMap list, so latency scales with the number of keys and the
// Kubernetes API server. Concurrency on a single key is handled by
// optimistic updates: a lost update surfaces as a conflict error to the
// caller rather than being retried.
//
// # Size comparison for 1kb (1024 byte) of data
//
// The table below shows how much of a key's own 1 MiB ConfigMap data budget
// a single 1kb value occupies per configuration. Sizes are the stored value
// size in the ConfigMap data map, not including the data key itself. They are
// verified by TestConfigMapKvStore_SizeComparisonTable against a highly
// compressible 1024 byte payload, which is representative for the store's
// intended use-cases:
//
//	Config                          Stored size for 1kb of data
//	plain (no compression, no encoding)   1024 B
//	base64 encoding only                   1368 B
//	json encoding only                     1034 B
//	csv encoding only                      1034 B
//	gzip level 1 + base64                   88 B
//	gzip level 9 + base64                   76 B
//
// Compression with base64 on top shrinks the compressible payload to well
// under 10% of its plain size; the encoding-only options never shrink it
// (base64 grows it, json and csv only add a small framing overhead). Because
// each key owns its ConfigMap, even a plain 1kb value leaves the full 1 MiB
// (1048576 B) budget of that key's ConfigMap available.
//
// A key's ConfigMap is created on first write. Reads and existence checks
// against a key whose ConfigMap does not exist yet report the key as missing
// (false), and deleting a key removes its ConfigMap.
//
// # Namespace
//
// The store's ConfigMaps live in the namespace the application currently runs
// in: inside a pod that is the pod's namespace, read from the namespace file
// of the mounted service account (the same service account the in cluster
// client authenticates with); outside a pod it is the namespace of the
// kubeconfig's current context. An explicit namespace in the store
// configuration overrides the resolved one, so a store can also target any
// other namespace the service account (or kubeconfig) user has access to.

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/encoding/json"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/refl"
)

const (
	// ConfigMapKeyPrefix is the fixed prefix of every ConfigMap name and data
	// key created by the ConfigMap backed KvStore.
	ConfigMapKeyPrefix = "kvstore"
	// ConfigMapMaxKeyLength is the maximum length of Kubernetes spec names
	// (RFC 1123 labels). It bounds every ConfigMap name this store generates,
	// keeping each name within a single DNS label so it is safe for all
	// consumers even though object names could technically be longer.
	ConfigMapMaxKeyLength = 63
	// ConfigMapMaxDataSize is the maximum size in bytes of a ConfigMap's data
	// map as enforced by the Kubernetes API server. Because the store uses one
	// ConfigMap per key, this limit applies to a single key's value.
	ConfigMapMaxDataSize = 1024 * 1024

	// ConfigMapCompressionGzip is the gzip compression algorithm. It is the
	// only algorithm supported by the ConfigMap backed KvStore today.
	ConfigMapCompressionGzip = "gzip"
)

// compressionLevels is the set of valid gzip compression levels as accepted
// by compress/gzip. The valid levels are gzip.DefaultCompression (-1),
// gzip.NoCompression (0), gzip.BestSpeed (1) and gzip.BestCompression (9).
var compressionLevels = []int{
	gzip.NoCompression,
	gzip.BestSpeed,
	gzip.BestCompression,
	gzip.DefaultCompression,
}

// CompressionSettings carries the compression settings of a ConfigMap backed
// KvStore. When Enabled is false the remaining fields are ignored.
//
// The compressed bytes are returned as raw binary by the compress and
// decompress helpers. Making those bytes safe for a ConfigMap's string data
// is the responsibility of the encoding layer (e.g. base64), which must be
// applied on top of the compressed bytes. This type deliberately does not do
// any encoding itself, so the two concerns stay independently testable.
type CompressionSettings struct {
	// Enabled turns compression on or off.
	Enabled bool `cfg:"enabled"`
	// Algo selects the compression algorithm. Supported values are listed in
	// SupportedCompressionAlgos; ConfigMapCompressionGzip is the only one
	// implemented today.
	Algo string `cfg:"algo"`
	// Level is the algorithm specific compression level. For gzip the valid
	// values are gzip.NoCompression (0), gzip.BestSpeed (1),
	// gzip.BestCompression (9) and gzip.DefaultCompression (-1). The level is
	// only evaluated when Enabled is true and Algo is valid.
	Level int `cfg:"level"`
}

// SupportedCompressionAlgos returns the compression algorithms the ConfigMap
// backed KvStore accepts in CompressionSettings.Algo.
func SupportedCompressionAlgos() []string {
	return []string{ConfigMapCompressionGzip}
}

// ValidCompressionLevels returns the compression levels accepted for the
// given algorithm, ordered ascending. It returns nil for unknown algorithms.
func ValidCompressionLevels(algo string) []int {
	switch algo {
	case ConfigMapCompressionGzip:
		levels := append([]int(nil), compressionLevels...)
		sort.Ints(levels)

		return levels
	default:
		return nil
	}
}

// CompressionAlgo returns the effective compression algorithm: the configured
// one when compression is enabled, an empty string when it is disabled.
func (s *CompressionSettings) CompressionAlgo() string {
	if s == nil || !s.Enabled {
		return ""
	}

	return s.Algo
}

// validateCompressionSettings validates the compression settings and returns
// descriptive errors for unknown algorithms and out-of-range levels. It
// accepts nil settings (compression disabled) and settings with Enabled false
// without error, because disabled compression leaves the other fields unused.
func validateCompressionSettings(settings *CompressionSettings) error {
	if settings == nil || !settings.Enabled {
		return nil
	}

	supported := SupportedCompressionAlgos()
	if !funk.Contains(supported, settings.Algo) {
		return fmt.Errorf(
			"invalid compression algorithm %q: supported compression algorithms are %s",
			settings.Algo, strings.Join(supported, ", "),
		)
	}

	levels := ValidCompressionLevels(settings.Algo)
	for _, level := range levels {
		if settings.Level == level {
			return nil
		}
	}

	return fmt.Errorf(
		"invalid compression level %d for algorithm %q: valid levels are %v",
		settings.Level, settings.Algo, levels,
	)
}

// compressValue compresses the given bytes according to the compression
// settings. Disabled or nil settings return the bytes unchanged. The result
// is raw compressed binary; making it safe for a ConfigMap string data entry
// is the job of the encoding layer (see CompressionSettings).
//
// Compression errors are never returned silently: a failed compression
// returns an error instead of writing uncompressed data, so a store can not
// end up with half-compressed values that a reader would misinterpret.
func compressValue(settings *CompressionSettings, data []byte) ([]byte, error) {
	if settings == nil || !settings.Enabled {
		return data, nil
	}

	if err := validateCompressionSettings(settings); err != nil {
		return nil, err
	}

	switch settings.Algo {
	case ConfigMapCompressionGzip:
		return gzipBytes(data, settings.Level)
	default:
		// unreachable: validateCompressionSettings rejects unknown algorithms
		return nil, fmt.Errorf("unsupported compression algorithm %q", settings.Algo)
	}
}

// decompressValue is the inverse of compressValue. Disabled or nil settings
// return the bytes unchanged, so values written without compression are read
// back as-is.
func decompressValue(settings *CompressionSettings, data []byte) ([]byte, error) {
	if settings == nil || !settings.Enabled {
		return data, nil
	}

	if err := validateCompressionSettings(settings); err != nil {
		return nil, err
	}

	switch settings.Algo {
	case ConfigMapCompressionGzip:
		return gunzipBytes(data)
	default:
		// unreachable: validateCompressionSettings rejects unknown algorithms
		return nil, fmt.Errorf("unsupported compression algorithm %q", settings.Algo)
	}
}

// gzipBytes compresses data with the given gzip level. gzip.DefaultCompression
// (-1) and any other level are passed through to the library, which treats
// invalid levels as the default; valid levels are 0 (no compression) to 9
// (best compression).
func gzipBytes(data []byte, level int) ([]byte, error) {
	var buf bytes.Buffer

	w, err := gzip.NewWriterLevel(&buf, level)
	if err != nil {
		return nil, fmt.Errorf("can not create gzip writer for level %d: %w", level, err)
	}

	if _, err := w.Write(data); err != nil {
		return nil, fmt.Errorf("can not gzip data: %w", err)
	}

	// Close flushes the gzip trailer; a truncated stream without it would fail
	// to decompress.
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("can not finish gzip stream: %w", err)
	}

	return buf.Bytes(), nil
}

// gunzipBytes decompresses gzip encoded data. The result is a fresh slice;
// callers may modify it without affecting the input.
func gunzipBytes(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("can not create gzip reader: %w", err)
	}

	out, readErr := io.ReadAll(r)
	// Close after the read so the gzip trailer and checksum are fully
	// consumed; a truncated or corrupt stream fails here.
	closeErr := r.Close()

	if readErr != nil {
		return nil, fmt.Errorf("can not gunzip data: %w", readErr)
	}

	if closeErr != nil {
		return nil, fmt.Errorf("can not close gzip reader: %w", closeErr)
	}

	return out, nil
}

const (
	// ConfigMapEncodingCSV is the csv encoding format. Values are stored as a
	// single-row CSV document, see csvBytes and csvDecodeString for the exact
	// representation.
	ConfigMapEncodingCSV = "csv"
	// ConfigMapEncodingJSON is the json encoding format. Values are stored as
	// their compact JSON representation.
	ConfigMapEncodingJSON = "json"
	// ConfigMapEncodingBase64 is the base64 encoding format. Values are stored
	// as standard base64 (RFC 4648) and are the only format that can carry
	// arbitrary bytes losslessly, so it is required when compression is
	// enabled.
	ConfigMapEncodingBase64 = "base64"
)

// EncodingSettings carries the encoding settings of a ConfigMap backed
// KvStore. When Enabled is false the remaining fields are ignored.
//
// The encoded value is a plain string and is safe to store in a
// ConfigMap's data map. Making raw (optionally compressed) binary safe for
// string storage is the job of this layer; making values small is the job
// of the compression layer. The two are independent and composable.
type EncodingSettings struct {
	// Enabled turns encoding on or off.
	Enabled bool `cfg:"enabled"`
	// Format selects the encoding format. Supported values are listed in
	// SupportedEncodingFormats.
	Format string `cfg:"format"`
}

// SupportedEncodingFormats returns the encoding formats the ConfigMap backed
// KvStore accepts in EncodingSettings.Format.
func SupportedEncodingFormats() []string {
	return []string{ConfigMapEncodingCSV, ConfigMapEncodingJSON, ConfigMapEncodingBase64}
}

// EncodingFormat returns the effective encoding format: the configured one
// when encoding is enabled, an empty string when it is disabled.
func (s *EncodingSettings) EncodingFormat() string {
	if s == nil || !s.Enabled {
		return ""
	}

	return s.Format
}

// validateSettingsCombination enforces the constraint between the compression
// and encoding layers: compressed bytes are raw binary and only the base64
// format can carry them losslessly in a ConfigMap's string data. csv would
// reject or mangle them and json is lossy for arbitrary bytes, so compression
// is only valid in combination with an enabled base64 encoding.
func validateSettingsCombination(compression *CompressionSettings, encoding *EncodingSettings) error {
	if compression == nil || !compression.Enabled {
		return nil
	}

	if encoding == nil || !encoding.Enabled {
		return fmt.Errorf(
			"compression is enabled but encoding is disabled: compression requires encoding with format %q, set Encoding to &EncodingSettings{Enabled: true, Format: %q}",
			ConfigMapEncodingBase64, ConfigMapEncodingBase64,
		)
	}

	if encoding.Format != ConfigMapEncodingBase64 {
		return fmt.Errorf(
			"compression is enabled but the encoding format is %q: compression requires the %q encoding format because compressed bytes are raw binary and only base64 can store them losslessly",
			encoding.Format, ConfigMapEncodingBase64,
		)
	}

	return nil
}

// validateEncodingSettings validates the encoding settings and returns a
// descriptive error for unknown formats. It accepts nil settings (encoding
// disabled) and settings with Enabled false without error, because disabled
// encoding leaves the other fields unused.
func validateEncodingSettings(settings *EncodingSettings) error {
	if settings == nil || !settings.Enabled {
		return nil
	}

	supported := SupportedEncodingFormats()
	if !funk.Contains(supported, settings.Format) {
		return fmt.Errorf(
			"invalid encoding format %q: supported encoding formats are %s",
			settings.Format, strings.Join(supported, ", "),
		)
	}

	return nil
}

// encodeValue encodes the given bytes according to the encoding settings.
// Disabled or nil settings return the bytes unchanged. The result is always
// a string-safe value that can be stored directly in a ConfigMap data entry.
//
// Encoding errors are never returned silently: a failed encoding returns an
// error instead of storing a half-encoded value that a reader would
// misinterpret.
func encodeValue(settings *EncodingSettings, data []byte) (string, error) {
	if settings == nil || !settings.Enabled {
		return string(data), nil
	}

	if err := validateEncodingSettings(settings); err != nil {
		return "", err
	}

	switch settings.Format {
	case ConfigMapEncodingBase64:
		return base64.StdEncoding.EncodeToString(data), nil
	case ConfigMapEncodingJSON:
		return jsonEncodeBytes(data)
	case ConfigMapEncodingCSV:
		return csvBytes(data)
	default:
		// unreachable: validateEncodingSettings rejects unknown formats
		return "", fmt.Errorf("unsupported encoding format %q", settings.Format)
	}
}

// decodeValue is the inverse of encodeValue. Disabled or nil settings return
// the bytes unchanged, so values stored without encoding are read back as-is.
func decodeValue(settings *EncodingSettings, raw string) ([]byte, error) {
	if settings == nil || !settings.Enabled {
		return []byte(raw), nil
	}

	if err := validateEncodingSettings(settings); err != nil {
		return nil, err
	}

	switch settings.Format {
	case ConfigMapEncodingBase64:
		out, err := base64.StdEncoding.DecodeString(raw)
		if err != nil {
			return nil, fmt.Errorf("can not base64 decode value: %w", err)
		}

		return out, nil
	case ConfigMapEncodingJSON:
		return jsonDecodeBytes(raw)
	case ConfigMapEncodingCSV:
		return csvDecodeString(raw)
	default:
		// unreachable: validateEncodingSettings rejects unknown formats
		return nil, fmt.Errorf("unsupported encoding format %q", settings.Format)
	}
}

// jsonEncodeBytes encodes arbitrary bytes as a JSON string value. A JSON
// string can carry any byte sequence: control characters, unicode and
// invalid utf-8 are escaped by the encoder.
func jsonEncodeBytes(data []byte) (string, error) {
	out, err := json.Marshal(string(data))
	if err != nil {
		return "", fmt.Errorf("can not json encode value: %w", err)
	}

	return string(out), nil
}

// jsonDecodeBytes is the inverse of jsonEncodeBytes. The stored value must be
// a JSON string value; anything else is rejected so a json-encoded entry is
// never confused with a plain json value stored without encoding. Note that
// json.Unmarshal alone does not catch non-string input: it leaves the target
// untouched without an error, so the leading quote is checked explicitly.
func jsonDecodeBytes(raw string) ([]byte, error) {
	if !strings.HasPrefix(strings.TrimSpace(raw), `"`) {
		return nil, fmt.Errorf("can not json decode value: expected a json string value, got %q", raw)
	}

	var decoded string
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return nil, fmt.Errorf("can not json decode value (expected a json string value): %w", err)
	}

	return []byte(decoded), nil
}

// csvBytes encodes arbitrary bytes as a deterministic single-row CSV document
// consisting of exactly one field, so the value can be stored as a plain
// string in a ConfigMap data entry.
//
// The encoding uses the standard CSV quoting rules: the field is left bare
// when it contains no comma, double quote, LF or CR, otherwise it is wrapped
// in double quotes with embedded quotes doubled. Two constraints keep the
// representation lossless and deterministic:
//   - an empty value is stored as the quoted empty field "", because the
//     csv writer emits a bare newline for an empty record, which the csv
//     reader would not parse back;
//   - values containing a carriage return (CR) are rejected. The csv reader
//     normalizes CR LF sequences to LF, so a bare or trailing CR could not be
//     recovered from the stored document.
func csvBytes(data []byte) (string, error) {
	if len(data) == 0 {
		return `""`, nil
	}

	if strings.ContainsRune(string(data), '\r') {
		return "", fmt.Errorf("can not csv encode value: carriage return bytes are not supported by the csv format")
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	if err := w.Write([]string{string(data)}); err != nil {
		return "", fmt.Errorf("can not csv encode value: %w", err)
	}

	w.Flush()

	// the writer terminates the record with a trailing newline; strip it so
	// the stored entry is exactly the single-row document
	return strings.TrimSuffix(buf.String(), "\n"), nil
}

// csvDecodeString is the inverse of csvBytes. The stored value must parse as
// a single-row CSV document with exactly one field; anything else is an
// error, not silently misread.
func csvDecodeString(raw string) ([]byte, error) {
	r := csv.NewReader(strings.NewReader(raw))

	record, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("can not csv decode value: %w", err)
	}

	if len(record) != 1 {
		return nil, fmt.Errorf("can not csv decode value: expected a single field, got %d", len(record))
	}

	// the writer produces exactly one record; a stored entry with trailing
	// rows is corrupted or foreign data and must be rejected, not silently
	// truncated
	if _, err := r.Read(); err != io.EOF {
		return nil, fmt.Errorf("can not csv decode value: expected a single record, got more")
	}

	return []byte(record[0]), nil
}

// ConfigMapSettings carries the settings of a ConfigMap backed KvStore.
//
// Namespace is the namespace the store's ConfigMaps live in. It is resolved
// at store creation: the namespace the application currently runs in (the pod
// namespace of the service account in cluster, the kubeconfig's current
// context namespace outside a pod), with an explicit namespace from the store
// configuration taking precedence.
type ConfigMapSettings struct {
	Namespace   string
	StoreName   string
	BatchSize   int
	Compression *CompressionSettings
	Encoding    *EncodingSettings
}

// ConfigMapConfiguration is the configurable representation of the ConfigMap
// backed KvStore settings, loaded from the "configmap" section of a kvstore's
// configuration block. Compression and Encoding use value types (the
// configurable store only populates value, not pointer, nested structs); the
// NewConfigMapKvStore factory turns them into the nil-means-disabled pointer
// settings of ConfigMapSettings based on their Enabled flag.
//
// Namespace is the namespace the store's ConfigMaps live in. When empty the
// store uses the namespace the application currently runs in (see
// ConfigMapSettings.Namespace), so the setting is optional and only needed to
// override that default.
type ConfigMapConfiguration struct {
	Namespace   string              `cfg:"namespace"`
	Compression CompressionSettings `cfg:"compression"`
	Encoding    EncodingSettings    `cfg:"encoding"`
}

// toSettings converts the configurable value-typed compression and encoding
// into the nil-means-disabled pointer settings of ConfigMapSettings.
func (c ConfigMapConfiguration) toSettings(storeName string, batchSize int) *ConfigMapSettings {
	var compression *CompressionSettings
	if c.Compression.Enabled {
		compression = &c.Compression
	}

	var encoding *EncodingSettings
	if c.Encoding.Enabled {
		encoding = &c.Encoding
	}

	return &ConfigMapSettings{
		Namespace:   c.Namespace,
		StoreName:   storeName,
		BatchSize:   batchSize,
		Compression: compression,
		Encoding:    encoding,
	}
}

type configMapKvStore[T any] struct {
	client    kubernetes.Interface
	namespace string
	storeName string
	settings  *ConfigMapSettings
}

// NewConfigMapKvStore is the ElementFactory of the ConfigMap backed KvStore
// as used by the configurable store and the chain store. It loads the
// store specific settings from the "configmap" section of the kvstore's
// configuration block, builds the kubernetes client and creates the store.
// The store name is taken from the kvstore name itself. Like the other
// element factories it returns the store wrapped in a metrics store when
// metrics are enabled on the passed settings.
//
// The store's namespace is resolved from the environment: an explicit
// namespace in the store configuration wins; otherwise the namespace the
// application currently runs in is used (see resolveKubernetesNamespace).
func NewConfigMapKvStore[T any](ctx context.Context, config cfg.Config, logger log.Logger, settings *Settings) (KvStore[T], error) {
	if reflect.ValueOf(new(T)).Elem().Kind() == reflect.Pointer {
		return nil, fmt.Errorf("the generic type T should not be a pointer type but is of type %T", *new(T))
	}

	configuration := ConfigMapConfiguration{}
	if err := config.UnmarshalKey(fmt.Sprintf("%s.configmap", GetConfigurableKey(settings.Name)), &configuration); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configmap kvstore configuration for %s: %w", settings.Name, err)
	}

	client, err := newKubernetesClient(ctx, config, logger, settings.Name)
	if err != nil {
		return nil, fmt.Errorf("can not create kubernetes client for configmap kvstore %s: %w", settings.Name, err)
	}

	namespace, err := resolveKubernetesNamespace(ctx, logger, settings.Name, configuration.Namespace)
	if err != nil {
		return nil, fmt.Errorf("can not resolve namespace for configmap kvstore %s: %w", settings.Name, err)
	}

	store, err := NewConfigMapKvStoreWithClient[T](client, namespace, settings.Name, configuration.toSettings(settings.Name, settings.BatchSize))
	if err != nil {
		return nil, err
	}

	return NewMetricStoreWithInterfaces[T](store, settings), nil
}

// newKubernetesClient is the factory NewConfigMapKvStore uses to build the
// kubernetes client. It is a variable so tests can replace it with a fake
// client.
var newKubernetesClient = provideKubernetesClient

// provideKubernetesClient provides a cached kubernetes clientset. Inside a
// pod the in cluster configuration is used, outside a pod the kubeconfig is
// resolved with the standard clientcmd loading rules (KUBECONFIG env var,
// then ~/.kube/config).
func provideKubernetesClient(ctx context.Context, _ cfg.Config, logger log.Logger, name string) (kubernetes.Interface, error) {
	key := fmt.Sprintf("kvstore.configmap.client.%s", name)

	return appctx.Provide(ctx, key, func() (kubernetes.Interface, error) {
		if restConfig, err := rest.InClusterConfig(); err == nil {
			logger.Debug(ctx, "creating kubernetes client for configmap kvstore %s from in cluster config", name)

			return kubernetes.NewForConfig(restConfig)
		}

		restConfig, err := restConfigFromKubeconfig()
		if err != nil {
			return nil, fmt.Errorf("can not create kubernetes client config: %w", err)
		}

		logger.Debug(ctx, "creating kubernetes client for configmap kvstore %s from kubeconfig", name)

		return kubernetes.NewForConfig(restConfig)
	})
}

// kubernetesServiceAccountDir is the directory the kubernetes service
// account of a pod is mounted in. The namespace file in it carries the pod's
// namespace; the token file in it is the same token rest.InClusterConfig
// uses, so a readable namespace file is a reliable in-cluster signal.
const kubernetesServiceAccountDir = "/var/run/secrets/kubernetes.io/serviceaccount"

// serviceAccountNamespaceFile is the path of the pod namespace file of the
// mounted service account. It is a variable so tests can point it at a
// temporary file.
var serviceAccountNamespaceFile = kubernetesServiceAccountDir + "/namespace"

// namespaceResolver resolves the namespace a ConfigMap backed KvStore should
// use. It is a variable so tests can replace it; by default it reads the
// in-cluster service account namespace and falls back to the kubeconfig's
// current context namespace.
var namespaceResolver = resolveNamespaceFromFilesystem

// resolveNamespaceFromFilesystem returns the namespace the application
// currently runs in: the pod namespace from the mounted service account when
// running inside a pod, the namespace of the kubeconfig's current context
// otherwise. It returns an error when neither source can provide a
// namespace, so a store is never silently created against a guessed
// namespace.
func resolveNamespaceFromFilesystem() (string, error) {
	if data, err := os.ReadFile(serviceAccountNamespaceFile); err == nil {
		if namespace := strings.TrimSpace(string(data)); namespace != "" {
			return namespace, nil
		}
	}

	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{})
	namespace, _, err := clientConfig.Namespace()
	if err != nil {
		return "", fmt.Errorf("can not determine current namespace: no in-cluster service account namespace and %w", err)
	}

	return namespace, nil
}

// restConfigFromKubeconfig builds a rest config from the standard kubeconfig
// loading rules.
func restConfigFromKubeconfig() (*rest.Config, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, &clientcmd.ConfigOverrides{})

	return clientConfig.ClientConfig()
}

// resolveKubernetesNamespace determines the namespace the ConfigMap backed
// KvStore for the given store uses: an explicit configured namespace wins,
// otherwise the namespace the application currently runs in is resolved
// (cached in the app context, see provideKubernetesClient for the client
// cache).
func resolveKubernetesNamespace(ctx context.Context, logger log.Logger, name, configured string) (string, error) {
	if configured != "" {
		logger.Debug(ctx, "configmap kvstore %s uses the configured namespace %s", name, configured)

		return configured, nil
	}

	namespace, err := appctx.Provide(ctx, "kvstore.configmap.namespace", func() (string, error) {
		namespace, err := namespaceResolver()
		if err != nil {
			return "", err
		}

		logger.Debug(ctx, "configmap kvstore %s uses the resolved namespace %s", name, namespace)

		return namespace, nil
	})
	if err != nil {
		return "", err
	}

	return namespace, nil
}

// NewConfigMapKvStoreWithClient creates a ConfigMap backed KvStore for the
// given namespace and store name using the provided client. Each key's
// ConfigMap is created on first write.
func NewConfigMapKvStoreWithClient[T any](client kubernetes.Interface, namespace, storeName string, settings *ConfigMapSettings) (KvStore[T], error) {
	if client == nil {
		return nil, fmt.Errorf("can not create configmap kvstore: client is required")
	}

	if err := validateNamespace(namespace); err != nil {
		return nil, fmt.Errorf("can not create configmap kvstore %q: %w", storeName, err)
	}

	if err := validateStoreName(storeName); err != nil {
		return nil, fmt.Errorf("can not create configmap kvstore %q: %w", storeName, err)
	}

	if settings == nil {
		settings = &ConfigMapSettings{}
	}

	if err := validateCompressionSettings(settings.Compression); err != nil {
		return nil, fmt.Errorf("can not create configmap kvstore %q: %w", storeName, err)
	}

	if err := validateEncodingSettings(settings.Encoding); err != nil {
		return nil, fmt.Errorf("can not create configmap kvstore %q: %w", storeName, err)
	}

	if err := validateSettingsCombination(settings.Compression, settings.Encoding); err != nil {
		return nil, fmt.Errorf("can not create configmap kvstore %q: %w", storeName, err)
	}

	return NewConfigMapKvStoreWithInterfaces[T](client, namespace, storeName, settings), nil
}

// NewConfigMapKvStoreWithInterfaces creates a ConfigMap backed KvStore from
// an already validated client, namespace and store name.
func NewConfigMapKvStoreWithInterfaces[T any](client kubernetes.Interface, namespace, storeName string, settings *ConfigMapSettings) KvStore[T] {
	return &configMapKvStore[T]{
		client:    client,
		namespace: namespace,
		storeName: storeName,
		settings:  settings,
	}
}

// configMapName returns the name of the ConfigMap that stores the given key:
// "kvstore-<storeName>-<keyName>". Each key owns its own ConfigMap.
func (s *configMapKvStore[T]) configMapName(keyStr string) string {
	return fmt.Sprintf("%s-%s-%s", ConfigMapKeyPrefix, s.storeName, keyStr)
}

// dataKey is the key under which the value is stored inside a key's
// dedicated ConfigMap. The ConfigMap is dedicated to a single key, so the
// key name itself is the data key.
func (s *configMapKvStore[T]) dataKey(keyStr string) string {
	return keyStr
}

// maxKeyNameLength is the longest keyName that still fits into the full
// ConfigMap name "kvstore-<storeName>-<keyName>" within the Kubernetes spec
// name length limit.
func (s *configMapKvStore[T]) maxKeyNameLength() int {
	return ConfigMapMaxKeyLength - len(ConfigMapKeyPrefix) - 2 - len(s.storeName)
}

// prepareStoredBytes is the write pipeline on bytes: compress according to
// the compression settings, then encode the (compressed) bytes into a
// string-safe representation according to the encoding settings. When both
// settings are disabled the bytes are returned as a string unchanged, so
// behavior is preserved for stores without compression or encoding.
func (s *configMapKvStore[T]) prepareStoredBytes(raw []byte) (string, error) {
	compressed, err := compressValue(s.settings.Compression, raw)
	if err != nil {
		return "", fmt.Errorf("can not compress value: %w", err)
	}

	stored, err := encodeValue(s.settings.Encoding, compressed)
	if err != nil {
		return "", fmt.Errorf("can not encode value: %w", err)
	}

	return stored, nil
}

// encodeForStorage is the write pipeline for a single value: marshal, then
// run the compressed/encoded storage preparation on the bytes.
func (s *configMapKvStore[T]) encodeForStorage(value T) (string, error) {
	raw, err := Marshal(value)
	if err != nil {
		return "", fmt.Errorf("can not marshal value: %w", err)
	}

	stored, err := s.prepareStoredBytes(raw)
	if err != nil {
		return "", err
	}

	return stored, nil
}

// decodeFromStorage is the inverse of encodeForStorage: decode the stored
// string back to bytes, decompress it according to the compression settings,
// then unmarshal it into the given value.
func (s *configMapKvStore[T]) decodeFromStorage(raw string, value any) error {
	decoded, err := decodeValue(s.settings.Encoding, raw)
	if err != nil {
		return fmt.Errorf("can not decode value: %w", err)
	}

	decompressed, err := decompressValue(s.settings.Compression, decoded)
	if err != nil {
		return fmt.Errorf("can not decompress value: %w", err)
	}

	if err := Unmarshal(decompressed, value); err != nil {
		return fmt.Errorf("can not unmarshal value: %w", err)
	}

	return nil
}

// key builds and validates the key name for the given key, rejecting empty
// keys and keys whose resulting ConfigMap name would exceed the Kubernetes
// spec name length limit.
func (s *configMapKvStore[T]) key(key any) (string, error) {
	keyStr, err := CastKeyToString(key)
	if err != nil {
		return "", fmt.Errorf("can not cast key %T %v to string: %w", key, key, err)
	}

	if keyStr == "" {
		return "", fmt.Errorf("invalid key name %q: key name must not be empty", keyStr)
	}

	maxLen := s.maxKeyNameLength()
	if len(keyStr) > maxLen {
		return "", fmt.Errorf(
			"invalid key name %q: length %d exceeds the maximum key name length %d (configmap name %q would exceed the kubernetes spec name length limit %d)",
			keyStr, len(keyStr), maxLen, s.configMapName(keyStr), ConfigMapMaxKeyLength,
		)
	}

	return keyStr, nil
}

func (s *configMapKvStore[T]) Contains(ctx context.Context, key any) (bool, error) {
	keyStr, err := s.key(key)
	if err != nil {
		return false, fmt.Errorf("can not get key to check in configmap: %w", err)
	}

	_, found, err := s.readKey(ctx, keyStr)
	if err != nil {
		return false, fmt.Errorf("can not check existence of key %s: %w", keyStr, err)
	}

	return found, nil
}

func (s *configMapKvStore[T]) Get(ctx context.Context, key any, value *T) (bool, error) {
	keyStr, err := s.key(key)
	if err != nil {
		return false, fmt.Errorf("can not get key to read from configmap: %w", err)
	}

	raw, found, err := s.readKey(ctx, keyStr)
	if err != nil {
		return false, fmt.Errorf("can not read configmap for key %s: %w", keyStr, err)
	}

	if !found {
		return false, nil
	}

	if err := s.decodeFromStorage(raw, value); err != nil {
		return false, fmt.Errorf("can not unmarshal value for key %s from configmap: %w", keyStr, err)
	}

	return true, nil
}

func (s *configMapKvStore[T]) GetBatch(ctx context.Context, keys any, values any) ([]any, error) {
	return getBatch(ctx, keys, values, s.getChunk, s.settings.BatchSize)
}

func (s *configMapKvStore[T]) getChunk(ctx context.Context, resultMap *refl.Map, keys []any) ([]any, error) {
	missing := make([]any, 0)

	for _, key := range keys {
		keyStr, err := s.key(key)
		if err != nil {
			return nil, fmt.Errorf("can not build key for key %T %v: %w", key, key, err)
		}

		raw, found, err := s.readKey(ctx, keyStr)
		if err != nil {
			return nil, fmt.Errorf("can not read configmap for key %s: %w", keyStr, err)
		}

		if !found {
			missing = append(missing, key)

			continue
		}

		element := resultMap.NewElement()
		if err := s.decodeFromStorage(raw, element); err != nil {
			return nil, fmt.Errorf("can not unmarshal value for key %s: %w", keyStr, err)
		}

		if err := resultMap.Set(key, element); err != nil {
			return nil, fmt.Errorf("can not set new element on result map for key %s: %w", keyStr, err)
		}
	}

	return missing, nil
}

func (s *configMapKvStore[T]) Put(ctx context.Context, key any, value T) error {
	keyStr, err := s.key(key)
	if err != nil {
		return fmt.Errorf("can not get key to write value to configmap: %w", err)
	}

	stored, err := s.encodeForStorage(value)
	if err != nil {
		return fmt.Errorf("can not prepare value for key %s: %w", keyStr, err)
	}

	return s.writeKey(ctx, keyStr, stored)
}

func (s *configMapKvStore[T]) PutBatch(ctx context.Context, values any) error {
	mii, err := refl.InterfaceToMapInterfaceInterface(values)
	if err != nil {
		return fmt.Errorf("could not convert values from %T to map[any]any: %w", values, err)
	}

	if len(mii) == 0 {
		return nil
	}

	// Encode every value up front so a single bad value fails the whole batch
	// before any ConfigMap is touched.
	stored := make(map[string]string, len(mii))
	for k, v := range mii {
		keyStr, err := s.key(k)
		if err != nil {
			return fmt.Errorf("can not get key to write value to configmap: %w", err)
		}

		raw, err := Marshal(v)
		if err != nil {
			return fmt.Errorf("can not marshal value for key %s: %w", keyStr, err)
		}

		encoded, err := s.prepareStoredBytes(raw)
		if err != nil {
			return fmt.Errorf("can not prepare value for key %s: %w", keyStr, err)
		}

		stored[keyStr] = encoded
	}

	for keyStr, encoded := range stored {
		if err := s.writeKey(ctx, keyStr, encoded); err != nil {
			return fmt.Errorf("can not write key %s to configmap: %w", keyStr, err)
		}
	}

	return nil
}

func (s *configMapKvStore[T]) Delete(ctx context.Context, key any) error {
	keyStr, err := s.key(key)
	if err != nil {
		return fmt.Errorf("can not get key to delete value from configmap: %w", err)
	}

	return s.deleteKey(ctx, keyStr)
}

func (s *configMapKvStore[T]) DeleteBatch(ctx context.Context, keys any) error {
	si, err := refl.InterfaceToInterfaceSlice(keys)
	if err != nil {
		return fmt.Errorf("could not convert keys from %T to []any: %w", keys, err)
	}

	seen := make(map[string]struct{}, len(si))
	for _, key := range si {
		keyStr, err := s.key(key)
		if err != nil {
			return fmt.Errorf("can not get key to delete value from configmap: %w", err)
		}

		seen[keyStr] = struct{}{}
	}

	for keyStr := range seen {
		if err := s.deleteKey(ctx, keyStr); err != nil {
			return fmt.Errorf("can not delete key %s from configmap: %w", keyStr, err)
		}
	}

	return nil
}

// EstimateSize returns the number of keys currently stored, i.e. the number
// of ConfigMaps in the namespace named "kvstore-<storeName>-*".
func (s *configMapKvStore[T]) EstimateSize() *int64 {
	size, err := s.countConfigMaps(context.Background())
	if err != nil {
		return nil
	}

	return mdl.Box(size)
}

// countConfigMaps lists the ConfigMaps in the namespace and counts the ones
// that belong to this store (name prefix "kvstore-<storeName>-").
func (s *configMapKvStore[T]) countConfigMaps(ctx context.Context) (int64, error) {
	configMaps, err := s.client.CoreV1().ConfigMaps(s.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return 0, fmt.Errorf("can not list configmaps in namespace %s: %w", s.namespace, err)
	}

	prefix := fmt.Sprintf("%s-%s-", ConfigMapKeyPrefix, s.storeName)
	size := int64(0)
	for _, configMap := range configMaps.Items {
		if strings.HasPrefix(configMap.Name, prefix) {
			size++
		}
	}

	return size, nil
}

// readKey reads the stored value of the given key from its dedicated
// ConfigMap. A key whose ConfigMap does not exist yet is reported as missing
// (found false, no error) so an absent key is never treated as a store error.
func (s *configMapKvStore[T]) readKey(ctx context.Context, keyStr string) (raw string, found bool, err error) {
	name := s.configMapName(keyStr)

	configMap, getErr := s.client.CoreV1().ConfigMaps(s.namespace).Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(getErr) {
		return "", false, nil
	}

	if getErr != nil {
		return "", false, fmt.Errorf("can not get configmap %s/%s: %w", s.namespace, name, getErr)
	}

	if configMap.Data == nil {
		return "", false, nil
	}

	value, ok := configMap.Data[s.dataKey(keyStr)]
	if !ok {
		return "", false, nil
	}

	return value, true, nil
}

// writeKey stores the given encoded value under the given key in its
// dedicated ConfigMap, creating the ConfigMap on first write. A concurrent
// modification surfaces as a conflict error to the caller.
func (s *configMapKvStore[T]) writeKey(ctx context.Context, keyStr, stored string) error {
	name := s.configMapName(keyStr)
	dataKey := s.dataKey(keyStr)
	data := map[string]string{dataKey: stored}

	if err := checkDataSize(data); err != nil {
		return err
	}

	existing, err := s.client.CoreV1().ConfigMaps(s.namespace).Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return s.createKeyConfigMap(ctx, name, data)
	}

	if err != nil {
		return fmt.Errorf("can not get configmap %s/%s for update: %w", s.namespace, name, err)
	}

	if existing.Data == nil {
		existing.Data = make(map[string]string, 1)
	}

	existing.Data[dataKey] = stored

	_, err = s.client.CoreV1().ConfigMaps(s.namespace).Update(ctx, existing, metav1.UpdateOptions{})
	if apierrors.IsConflict(err) {
		return fmt.Errorf("can not update configmap %s/%s: concurrent modification, %w", s.namespace, name, err)
	}

	if err != nil {
		return fmt.Errorf("can not update configmap %s/%s: %w", s.namespace, name, err)
	}

	return nil
}

// createKeyConfigMap creates the dedicated ConfigMap for a key from the given
// data. A concurrent create (already exists) is tolerated and re-read, so a
// lost create does not fail the write.
func (s *configMapKvStore[T]) createKeyConfigMap(ctx context.Context, name string, data map[string]string) error {
	_, err := s.client.CoreV1().ConfigMaps(s.namespace).Create(ctx, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: s.namespace,
		},
		Data: data,
	}, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("can not create configmap %s/%s: concurrent modification, %w", s.namespace, name, err)
	}

	if err != nil {
		return fmt.Errorf("can not create configmap %s/%s: %w", s.namespace, name, err)
	}

	return nil
}

// deleteKey removes the dedicated ConfigMap of the given key. A key whose
// ConfigMap does not exist yet is a no-op, so deleting an absent key is not
// an error.
func (s *configMapKvStore[T]) deleteKey(ctx context.Context, keyStr string) error {
	name := s.configMapName(keyStr)

	err := s.client.CoreV1().ConfigMaps(s.namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("can not delete configmap %s/%s: %w", s.namespace, name, err)
	}

	return nil
}

func checkDataSize(data map[string]string) error {
	size := 0
	for k, v := range data {
		size += len(k) + len(v)
	}

	if size > ConfigMapMaxDataSize {
		return fmt.Errorf(
			"can not update configmap: data map size %d exceeds the kubernetes configmap size limit %d",
			size, ConfigMapMaxDataSize,
		)
	}

	return nil
}

func validateNamespace(namespace string) error {
	if namespace == "" {
		return fmt.Errorf("namespace must not be empty")
	}

	if len(namespace) > ConfigMapMaxKeyLength {
		return fmt.Errorf("namespace %q exceeds the kubernetes spec name length limit %d", namespace, ConfigMapMaxKeyLength)
	}

	return nil
}

func validateStoreName(storeName string) error {
	if storeName == "" {
		return fmt.Errorf("store name must not be empty")
	}

	if maxLen := ConfigMapMaxKeyLength - len(ConfigMapKeyPrefix) - 2; len(storeName) > maxLen {
		return fmt.Errorf(
			"store name %q exceeds the maximum store name length %d (configmap name %q would exceed the kubernetes spec name length limit %d)",
			storeName, maxLen, fmt.Sprintf("%s-%s", ConfigMapKeyPrefix, storeName), ConfigMapMaxKeyLength,
		)
	}

	return nil
}
