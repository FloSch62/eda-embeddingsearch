# EDA Embedding Search

A command-line tool that converts natural language queries into EQL (Event Query Language) statements for querying Nokia SR Linux (SRL) and Service Router Operating System (SROS) devices.

## Features

- 🔍 **Natural Language Search**: Convert human-readable queries to precise EQL statements
- 🚀 **Fast Semantic Search**: Embedding-based search with vector similarity
- 🎯 **Smart Query Construction**: Automatically builds WHERE, ORDER BY, LIMIT clauses
- 📊 **Multi-Platform Support**: Works with both SRL and SROS devices
- ⚡ **Performance Optimized**: Binary caching and inverted indexing for speed
- 🔧 **Zero Configuration**: Auto-downloads embeddings on first use

## Installation

### Download Pre-built Binaries

Download the appropriate binary for your platform from the [releases page](https://github.com/eda-labs/eda-embeddingsearch/releases):

- **Linux**: `embeddingsearch-linux-amd64`
- **macOS Intel**: `embeddingsearch-darwin-amd64`
- **macOS Apple Silicon**: `embeddingsearch-darwin-arm64`
- **Windows**: `embeddingsearch-windows-amd64.exe`

### Build from Source

```bash
git clone https://github.com/eda-labs/eda-embeddingsearch.git
cd eda-embeddingsearch
make build
```

Binaries will be created in the `bin/` directory for all platforms:
- Linux: `bin/linux/embeddingsearch`
- macOS Intel: `bin/darwin/embeddingsearch`
- macOS ARM64: `bin/darwin/embeddingsearch-arm64`
- Windows: `bin/win32/embeddingsearch.exe`

For other build options, run `make help`.

## Usage

### Basic Usage

```bash
# Simple query
embeddingsearch "show bgp neighbors"

# Query with specific platform
embeddingsearch -platform srl "interface statistics"

# JSON output
embeddingsearch -json "top 5 processes by memory"

# Verbose mode for debugging
embeddingsearch -verbose "ospf neighbors state"
```

### Example Queries

```bash
# Show commands
embeddingsearch "show system information"
embeddingsearch "show interface ethernet-1/1 statistics"

# Top N queries
embeddingsearch "top 10 processes by cpu usage"
embeddingsearch "top 5 interfaces by traffic"

# Filtered queries
embeddingsearch "bgp neighbors with state established"
embeddingsearch "interfaces where oper-state is up"

# Node-specific queries
embeddingsearch "show cpu on node leaf-1"
embeddingsearch "memory usage on spine nodes"
```

### Command-line Options

```
Usage: embeddingsearch [options] <query>

Options:
  -json              Output results in JSON format
  -platform string   Force platform type (srl or sros)
  -verbose           Enable verbose output for debugging
  -help              Show this help message
```

## How It Works

1. **Query Processing**: Your natural language query is tokenized and analyzed
2. **Semantic Search**: The tool searches through embeddings to find relevant database paths
3. **Query Construction**: Based on your input, it builds a complete EQL query with:
   - Field selection
   - WHERE conditions
   - ORDER BY clauses
   - LIMIT/DELTA for result control
4. **Platform Detection**: Automatically detects whether you're querying SRL or SROS

## Examples

### Simple Query
```bash
$ embeddingsearch "show interfaces"

Top match (score: 84.00):
.namespace.node.srl.acl.interface

Other possible matches:
1. .namespace.node.srl.interface (score: 84.00)
2. .namespace.node.srl.network-instance.interface (score: 74.00)
3. .namespace.node.srl.system.lldp.interface (score: 74.00)
...
```

### Top N with Ordering
```bash
$ embeddingsearch "top 5 processes by memory"

Top match (score: 50.00):
.namespace.node.srl.platform.control.memory order by [utilization descending] limit 5

Other possible matches:
1. .namespace.node.srl.platform.linecard.forwarding-complex.buffer-memory fields [used] order by [used descending] limit 5 (score: 23.50)
...
```

### BGP Query with Fields
```bash
$ embeddingsearch "bgp neighbors"

Top match (score: 62.50):
.namespace.node.srl.network-instance.protocols.bgp.neighbor fields [admin-state, session-state, last-state]
```

### JSON Output
```bash
$ embeddingsearch -json "interface ethernet-1/1"

{
  "topMatch": {
    "score": 75,
    "query": ".namespace.node.srl.interface",
    "table": ".namespace.node.srl.interface",
    "description": "The list of named interfaces on the device",
    "availableFields": [
      "admin-state",
      "description",
      "mtu",
      "oper-state",
      ...
    ]
  },
  "others": [...]
}
```

### Verbose Mode
```bash
$ embeddingsearch -v "bgp neighbors"

Top match (score: 62.50):
.namespace.node.srl.network-instance.protocols.bgp.neighbor fields [admin-state, session-state, last-state]

Query components:
  Table: .namespace.node.srl.network-instance.protocols.bgp.neighbor
  Fields: admin-state, session-state, last-state
```

## Advanced Features

### Synonym Expansion
The tool understands common networking terms and their variations:
- "bgp" → BGP, Border Gateway Protocol
- "ospf" → OSPF, Open Shortest Path First
- "cpu" → CPU, processor, processing

### Typo Tolerance
Common typos are automatically corrected:
- "interfcae" → "interface"
- "nieghbor" → "neighbor"
- "statistcs" → "statistics"

### Context-Aware Scoring
The search algorithm considers context:
- "show" commands prefer state paths over configuration
- "top N" queries automatically add sorting and limiting
- Platform-specific paths are prioritized

## Troubleshooting

### Embeddings Not Found
On first run, the tool automatically downloads embeddings. If this fails:

1. Check your internet connection
2. Verify you can access GitHub
3. Manually download from [embeddings repository](https://github.com/eda-labs/embeddings-library/releases)
4. Place files in `~/.eda/vscode/embeddings/`

### Platform Detection Issues
If the wrong platform is detected, use the `-platform` flag:
```bash
embeddingsearch -platform sros "show card detail"
```

### Debugging
Use `-verbose` flag to see detailed search process:
```bash
embeddingsearch -verbose "your query here"
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built for the [Event-Driven Automation (EDA)](https://github.com/eda-labs) ecosystem
- Embeddings provided by [eda-labs/embeddings-library](https://github.com/eda-labs/embeddings-library)