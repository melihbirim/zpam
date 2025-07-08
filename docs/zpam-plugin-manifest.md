# ZPAM Plugin Manifest Specification

## Overview

The `zpam-plugin.yaml` file is the manifest that defines a ZPAM plugin's metadata, dependencies, and configuration. This file must be present in the root directory of every ZPAM plugin.

## File Format

```yaml
# zpam-plugin.yaml
manifest_version: "1.0"

# Plugin metadata
plugin:
  name: "openai-classifier"
  version: "1.2.0"
  description: "AI-powered spam detection using OpenAI GPT models"
  author: "ZPAM Team"
  homepage: "https://github.com/zpam-team/openai-classifier"
  repository: "https://github.com/zpam-team/openai-classifier"
  license: "MIT"
  
  # Plugin classification
  type: "ml_classifier"  # content_analyzer, reputation_checker, attachment_scanner, ml_classifier, external_engine, custom_rule_engine
  tags: ["ai", "openai", "gpt", "machine-learning"]
  
  # ZPAM compatibility
  min_zpam_version: "2.1.0"
  max_zpam_version: "3.0.0"  # optional
  
# Plugin entry point
main:
  binary: "./bin/openai-classifier"       # For compiled plugins
  script: "./scripts/classify.py"         # For script-based plugins
  library: "./lib/openai_classifier.so"   # For shared libraries
  
# Dependencies
dependencies:
  system:
    - "python3"
    - "curl"
  zpam_plugins: []  # Other ZPAM plugins this depends on
  external:
    - name: "openai"
      version: ">=1.0.0"
      package_manager: "pip"
      install_command: "pip install openai>=1.0.0"

# Configuration schema
configuration:
  api_key:
    type: "string"
    required: true
    description: "OpenAI API key"
    env_var: "OPENAI_API_KEY"
    sensitive: true
  
  model:
    type: "string"
    required: false
    default: "gpt-3.5-turbo"
    description: "OpenAI model to use"
    allowed_values: ["gpt-3.5-turbo", "gpt-4", "gpt-4o"]
  
  max_tokens:
    type: "integer"
    required: false
    default: 150
    description: "Maximum tokens for API responses"
    min: 10
    max: 1000

# Plugin interfaces implemented
interfaces:
  - "MLClassifier"
  - "ContentAnalyzer"  # Can implement multiple interfaces

# Security and validation
security:
  permissions:
    - "network_access"     # Plugin needs internet access
    - "file_read"         # Plugin reads files
    - "env_vars"          # Plugin reads environment variables
  
  sandbox: false           # Whether plugin should run in sandbox
  code_signing:
    enabled: true
    public_key: "-----BEGIN PUBLIC KEY-----\n...\n-----END PUBLIC KEY-----"

# Metadata for marketplace
marketplace:
  category: "AI & Machine Learning"
  keywords: ["spam", "ai", "openai", "classification"]
  screenshots:
    - "./docs/screenshot1.png"
    - "./docs/screenshot2.png"
  documentation: "./README.md"
  
# Build and packaging
build:
  dockerfile: "./Dockerfile"        # For containerized plugins
  build_script: "./build.sh"       # Custom build script
  install_script: "./install.sh"   # Custom installation
  
  # Output artifacts
  artifacts:
    - "./bin/openai-classifier"
    - "./config/default.yaml"
    - "./docs/"

# Testing
testing:
  test_command: "./test.sh"
  test_data: "./test-data/"
  benchmark_command: "./benchmark.sh"

# Plugin lifecycle hooks
hooks:
  pre_install: "./scripts/pre-install.sh"
  post_install: "./scripts/post-install.sh"
  pre_uninstall: "./scripts/pre-uninstall.sh"
  post_uninstall: "./scripts/post-uninstall.sh"
```

## Field Descriptions

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `manifest_version` | string | Manifest format version (currently "1.0") |
| `plugin.name` | string | Unique plugin identifier (lowercase, hyphens) |
| `plugin.version` | string | Semantic version (e.g., "1.2.0") |
| `plugin.description` | string | Brief description of plugin functionality |
| `plugin.author` | string | Plugin author or organization |
| `plugin.type` | string | Plugin type (see Plugin Types below) |
| `main` | object | Plugin entry point configuration |
| `interfaces` | array | List of ZPAM plugin interfaces implemented |

### Plugin Types

| Type | Description | Interface |
|------|-------------|-----------|
| `content_analyzer` | Analyzes email content for spam indicators | `ContentAnalyzer` |
| `reputation_checker` | Checks sender/domain/URL reputation | `ReputationChecker` |
| `attachment_scanner` | Scans email attachments | `AttachmentScanner` |
| `ml_classifier` | Machine learning classification | `MLClassifier` |
| `external_engine` | Integration with external services | `ExternalEngine` |
| `custom_rule_engine` | Custom rule evaluation | `CustomRuleEngine` |

### Configuration Types

| Type | Description | Additional Fields |
|------|-------------|-------------------|
| `string` | Text value | `default`, `allowed_values`, `pattern` |
| `integer` | Numeric value | `default`, `min`, `max` |
| `boolean` | True/false | `default` |
| `array` | List of values | `default`, `item_type` |
| `object` | Complex object | `properties` |

### Security Permissions

| Permission | Description |
|------------|-------------|
| `network_access` | Plugin needs internet connectivity |
| `file_read` | Plugin reads files from filesystem |
| `file_write` | Plugin writes files to filesystem |
| `env_vars` | Plugin accesses environment variables |
| `system_commands` | Plugin executes system commands |

## Example Manifests

### Simple Content Analyzer

```yaml
manifest_version: "1.0"

plugin:
  name: "keyword-detector"
  version: "1.0.0"
  description: "Simple keyword-based spam detection"
  author: "Community"
  type: "content_analyzer"
  tags: ["keywords", "simple"]
  min_zpam_version: "2.0.0"

main:
  script: "./keyword_detector.py"

interfaces:
  - "ContentAnalyzer"

configuration:
  spam_keywords:
    type: "array"
    required: true
    description: "List of spam keywords to detect"
    default: ["urgent", "free", "winner"]

security:
  permissions: []
  sandbox: true
```

### External Service Integration

```yaml
manifest_version: "1.0"

plugin:
  name: "virustotal-scanner"
  version: "2.1.0"
  description: "VirusTotal URL and attachment scanning"
  author: "Security Corp"
  type: "reputation_checker"
  tags: ["virustotal", "security", "scanning"]
  min_zpam_version: "2.1.0"

main:
  binary: "./bin/virustotal-scanner"

interfaces:
  - "ReputationChecker"
  - "AttachmentScanner"

dependencies:
  system:
    - "curl"

configuration:
  api_key:
    type: "string"
    required: true
    description: "VirusTotal API key"
    env_var: "VIRUSTOTAL_API_KEY"
    sensitive: true
  
  scan_urls:
    type: "boolean"
    default: true
    description: "Enable URL scanning"

security:
  permissions:
    - "network_access"
  sandbox: false
```

## GitHub Repository Structure

For GitHub-based plugin discovery, repositories should:

1. **Include the `zpam-plugin` topic** in repository settings
2. **Have `zpam-plugin.yaml` in the root directory**
3. **Follow this recommended structure:**

```
zpam-plugin-example/
├── zpam-plugin.yaml          # Plugin manifest
├── README.md                 # Documentation
├── LICENSE                   # License file
├── src/                      # Source code
│   ├── main.go               # Main plugin code
│   └── config.go             # Configuration handling
├── bin/                      # Compiled binaries (if applicable)
├── scripts/                  # Installation/build scripts
├── test/                     # Test files
├── docs/                     # Additional documentation
└── examples/                 # Usage examples
```

## Validation

The ZPAM plugin system will validate:

1. **Manifest syntax** - Valid YAML format
2. **Required fields** - All mandatory fields present
3. **Version compatibility** - Compatible with current ZPAM version
4. **Interface compliance** - Plugin implements declared interfaces
5. **Security requirements** - Permissions match actual needs
6. **Dependencies** - All dependencies available

## Best Practices

1. **Semantic versioning** - Use proper semver (major.minor.patch)
2. **Clear descriptions** - Write helpful descriptions and documentation
3. **Minimal permissions** - Request only needed security permissions
4. **Comprehensive testing** - Include test suites and examples
5. **Regular updates** - Keep dependencies and compatibility current
6. **Security first** - Follow secure coding practices

## Future Extensions

The manifest format is designed to be extensible. Future versions may add:

- Multi-language support
- Plugin update mechanisms
- Performance benchmarking data
- Community ratings and reviews
- Automated security scanning results 