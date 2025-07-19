# Homebrew Formula for MakeMCP

This directory contains the Homebrew formula for installing MakeMCP.

## Installation Options

### Option 1: Using Homebrew Tap (Recommended)

```bash
# Add the tap (once the tap is created)
brew tap T4cceptor/makemcp

# Install MakeMCP
brew install makemcp
```

### Option 2: Direct Formula Installation

```bash
# Install directly from the formula file
brew install --build-from-source packaging/homebrew/makemcp.rb
```

### Option 3: Development Installation

```bash
# For development/testing
brew install --HEAD packaging/homebrew/makemcp.rb
```

## Formula Details

- **Name**: makemcp
- **Dependencies**: Go (build-time only)
- **License**: MIT
- **Build**: Uses Go modules for dependency management

## Testing the Formula

```bash
# Test the formula locally
brew install --build-from-source packaging/homebrew/makemcp.rb
brew test makemcp
```

## Updating the Formula

When releasing a new version:

1. Update the version in the URL
2. Update the SHA256 checksum (can be generated during release)
3. Test the formula locally
4. Submit to homebrew-core or update the tap

## Creating a Homebrew Tap

To create your own tap repository:

```bash
# Create a new repository named homebrew-makemcp
# Add this formula to Formula/makemcp.rb in that repository
```

Users can then install with:
```bash
brew tap T4cceptor/makemcp
brew install makemcp
```