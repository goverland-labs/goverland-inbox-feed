# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres
to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
