# lua-spam-keywords

ZPAM plugin for content-analyzer.

## Description

This plugin implements content-analyzer functionality for the ZPAM spam detection system.

## Installation

```bash
zpam plugins install github:yourusername/lua-spam-keywords
```

## Configuration

Edit your ZPAM configuration to enable this plugin:

```yaml
plugins:
  lua-spam-keywords:
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
