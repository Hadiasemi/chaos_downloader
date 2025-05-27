# Chaos Downloader

A simple tool to download, organize, and consolidate data from Project Discovery's Chaos dataset.
```bash

Chaos Data Downloader - Download and process company data from Project Discovery

USAGE:
    chaos_downloader [OPTIONS]

OPTIONS:
    -c string    Comma-separated list of company names to download
                 Example: -c "Tesla,Google,Microsoft"
    -i string    Path to file containing company names (one per line)
    -a           Download all available companies
    -h           Show this help message

EXAMPLES:
    # Download specific companies
    chaos_downloader -c Tesla
    chaos_downloader -c "Tesla,Google,Microsoft"

    # Download companies from file
    chaos_downloader -i companies.txt

    # Download all available companies
    chaos_downloader -a

DESCRIPTION:
    This tool downloads chaos data from Project Discovery for specified companies.
    Downloaded data is extracted to ./AllChaosData/ and all .txt files are
    concatenated into everything.txt in the current directory.
```
- - -
## Getting Started

Ensure Go is installed on your system to run this script.
- - -
### Steps

1. Clone this repository:

```
git clone https://github.com/tonytsep/chaos_downloader.git
```

2. Navigate to the repository folder:

```
cd chaos_downloader
```

3. Execute the script:

```
go run chaos_downloader.go
```

The script downloads ZIP files listed in https://chaos-data.projectdiscovery.io/index.json, extracts them into named directories, and compiles text from those directories into a single file named `everything.txt`.
- - -

## Usage

Running the script processes the data and generates `everything.txt` in the script's execution directory, alongside a folder `AllChaosData` containing the organized data.
- - -

## Contributing

Feel free to fork, modify, or suggest improvements to this script. Any contribution is welcome.
- - -

## License

This project is open-source, licensed under the MIT License.
