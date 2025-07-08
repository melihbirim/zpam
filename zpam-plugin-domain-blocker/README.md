# domain-blocker

ZPAM plugin for content-analyzer.

## Description

This plugin implements content-analyzer functionality for the ZPAM spam detection system.

## Installation

```bash
zpam plugins install github:yourusername/domain-blocker
```

## Configuration

Edit your ZPAM configuration to enable this plugin:

```yaml
plugins:
  domain-blocker:
    enabled: true
    weight: 1.0
    settings:
      example_setting: "your_value"
```

## Development

### Building

```bash
make build
```

### Testing

```bash
make test
```

### Publishing

```bash
zpam plugins publish
```

## License

MIT License - see LICENSE file for details.
