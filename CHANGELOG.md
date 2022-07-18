# Changelog
## [2.1.0] - 2022-07-18
### Added
### Changed
 - Use common base image for all NE components
## [2.0.4] - 2021-08-16
### Added
### Changed
 - default peripheral name
 - fix for unknown serial number
## [2.0.3] - 2021-08-12
### Added
### Changed
 - fix access to product name
## [2.0.2] - 2021-08-10
### Added
### Changed
 - fixed defer execution on condition
## [2.0.1] - 2021-07-28
### Added 
 - graceful shutdown in case on unsupported host
### Changed
## [2.0.0] - 2021-07-26
### Added 
 - full rewrite in Golang
### Changed
## [1.4.1] - 2021-01-07
### Added
### Changed
 - fix error parsing on agent api unavailable
## [1.4.0] - 2021-01-05
### Added
### Changed
 - removed bash client for Nuvla API - using local agent API instead
## [1.3.2] - 2020-12-11
        ### Added
        ### Changed
                  - disable TLS verification for edits and deletes of usb peripherals
## [1.3.1] - 2020-12-11
        ### Added
        ### Changed
                  - disable TLS verification for usb peripherals
## [1.3.0] - 2020-12-04
        ### Added 
                  - ca-certificates for secure communication with Nuvla 
                  - setup needed environment from shared storage volume
        ### Changed
                  - minor bug fixes and reduced logging noise
## [1.2.0] - 2020-11-26
        ### Added 
                  - reporting of disk capacity for USB flash drives
        ### Changed
## [1.1.2] - 2020-11-10
### Added
### Changed
- fixed existing USB devices check, for bug with conflict between files and directories in peripherals folders
## [1.1.1] - 2020-10-02
### Added 
- ONBUILD SixSq license dump
### Changed
## [1.1.0] - 2020-07-28
### Added 
- double checking and update of existing USB peripherals on every restart
### Changed
- parallelize USB device info gathering
## [1.0.4] - 2020-07-01
### Added
### Changed
- better management of USB peripheral on reboot or restart
- speed up API calls to Nuvla
- better handling of failed deletion of Nuvla resources
## [1.0.3] - 2020-03-13
### Added 
- new peripheral attributes 
- mapping from video peripheral to video device in the filesystem
### Changed
## [1.0.2] - 2020-01-03
### Added
### Changed
- fixed Nuvla endpoint parsing
- updated packg dependencies
## [1.0.1] - 2019-11-19
### Added
### Changed
- update packages
## [1.0.0] - 2019-07-03
### Added
  - auto discovery, classification and management of created and removed usb peripherals, in real time

