# lua-domain-checker

ZPAM plugin for reputation-checker.

## Description

This plugin implements reputation-checker functionality for the ZPAM spam detection system.

## Installation

```bash
zpam plugins install github:yourusername/lua-domain-checker
```

## Configuration

Edit your ZPAM configuration to enable this plugin:

```yaml
plugins:
  lua-domain-checker:
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
