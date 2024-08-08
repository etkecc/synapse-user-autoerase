# Synapse User Auto Eraser

This is a simple script that will erase users from a Synapse server based on specific criteria:

- not admin
- not guest
- not deactivated
- not contain a specific string in their MXID
- not older than a specific threshold in days

It was developed for [etke.cc demo server](https://etke.cc/demo/) for the purpose of cleaning up the user database.

Another purpose of this repo is to be a test stand for migrating our [gitlab repos](https://gitlab.com/etke.cc) to [github](https://github.com/etke-cc)

Don't expect this to be a full-featured tool, it's just a simple script that does one thing.

## Configuration

Configuration is done via environment variables (`.env` file is supported as well). The following variables are supported:

- `SUAE_HOST` - Synapse server host, e.g. `https://matrix.example.com` (without trailing slash)
- `SUAE_TOKEN` - Synapse homeserver admin token
- `SUAE_PREFIXES` - Space-separated list of additional MXID prefixes that should be excluded from deletion
- `SUAE_TTL` - Maximum age of the user in days, users younger than this value will not be deleted
- `SUAE_DRYRUN` - If set to `true`, the script will only print the list of users that would be deleted. You **WANT** to run it in dry-run mode first to make sure you're not deleting the wrong users.

Check `.env.example` for an example configuration.


## Usage

```bash
# if you have binary, run it like this
$ suae
# if you want to run from source
$ just run
```
