# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres
to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.1] - 2024-11-01

### Fixed
- Fixed go version in dockerfile and github actions

## [0.2.0] - 2024-11-01

### Added
- Added github actions
- Added README.md
- Added CONTRIBUTING.md
- Added LICENSE

## [0.1.3] - 2024-08-23

### Fixed
- Skip processed subscribers on feed

## [0.1.2] - 2024-08-19

### Fixed
- Update proposals for unsubscribed users

## [0.1.1] - 2024-07-25

### Changed
- Do not auto archive feed items marked as unarchived by user

## [0.1.0] - 2024-07-05

### Added
- Autoarchive feed item after voting on proposal
- Consume settings updates
- Autoarchive after N days by user settings

## [0.0.22] - 2024-04-02

### Changed
- Don’t fill Feed with completed items

## [0.0.21] - 2024-03-11

### Fixed
- Fixed calc counters for mark as read/unread endpoints

## [0.0.20] - 2024-03-06

### Changed
- Update platform events library to collect nats metrics

## [0.0.19] - 2024-02-13

### Added
- Added total and unread counters for methods with changing read state

## [0.0.18] - 2024-02-09

### Changed
- Changed unread endpoints

## [0.0.17] - 2024-02-08

### Added
- Filtering inbox by spam proposals
- Filtering inbox by closed proposals

## [0.0.16] - 2024-02-05

### Removed
- Moved preparations for pushes to inbox-push service

## [0.0.15] - 2023-12-29

### Changed
- Decrease 1 week param to 1 day param for auto-archiving 
- Do not add to inbox inactive proposals

## [0.0.14] - 2023-12-21

### Changed
- Improve auto archiving logic

## [0.0.13] - 2023-12-14

### Added
- Basic auto archive logic

## [0.0.12] - 2023-12-06

### Changed
- Update feed sorting
- Mark everything as read unless params specified

## [0.0.11] - 2023-11-10

### Changed
- Updating updated_at column

## [0.0.10] - 2023-11-10

### Changed
- Sorting from created_at to updated_at
- Mark read by updated_at field

## [0.0.9] - 2023-11-07

### Changed
- Add reactive subscription on DAO

## [0.0.8] - 2023-11-07

### Changed
- Do not prefill feed on empty subscriptions

## [0.0.7] - 2023-10-10

### Added
- Added option for fetching archived items only

### Fixed
- Fixed prefilling feed if it isn't required

## [0.0.6] - 2023-09-18

### Changed
- Update push title for the ProposalVotingEndsSoon event

## [0.0.5] - 2023-09-18

### Changed
- Prefill logic based on new requirements

## [0.0.4] - 2023-08-25

### Changed
- Sort feed timeline based on action weight
- Correct created_at time based on the earliest event date
- Correct quorum reached time based on finished at voting 

### Added
- Send push message to the queue

## [0.0.3] - 2023-07-26

### Changed
- Mark unread only on timeline change

## [0.0.2] - 2023-07-26

### Added
- Added prefill feed for empty subscribers

### Fixed
- Fixed ordering of feed items

## [0.0.1] - 2023-07-15

### Added
- First version
