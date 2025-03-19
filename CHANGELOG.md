# Changelog

All notable changes to this project will be documented in this file.

## 0.1.2 - 2023-07-15

### Added

- One-line installation script (`install.sh`) for easy installation
- Command line arguments support (`--help`, `--version`)
- Improved shell detection in installation script
- Support for various shells (zsh, bash) and automatic PATH configuration
- Comprehensive README with usage examples and configuration guides

### Changed

- Updated installation process to use `~/.local/bin` instead of system directories
- Improved error handling in startup process
- Enhanced logging for better debugging

### Fixed

- Fixed issue with relative paths in compression tools
- Improved error messages for failed compressions
- Better handling of invalid file formats

## 0.1.1 - 2023-06-20

### Changed

- Updated version number in application metadata
- Minor performance improvements

### Fixed

- Fixed compatibility issue with macOS 14.0
- Corrected parameter validation for advanced compression

## 0.1.0 - 2023-06-05

### Added

- Initial release
- Support for quick image compression
- Support for advanced image compression with customizable parameters
- Multiple output format support (JPEG, WebP, HEIC, AVIF, PNG)
- Batch processing capability
