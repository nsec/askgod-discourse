## Installation

1. `sudo apt-get install -y libsqlite3-dev`
2. `make linux`
3. `cp config.yaml.example config.yaml`
4. Fill `config.yaml` with the appropriate values.
5. `openssl s_client -connect DISCOURSE_DOMAIN.nsec:443 -showcerts`. Clean up to keep only the first certificate. Put it in `config.yaml` under `discourse_cert`.
6. `openssl s_client -connect ASKGOD_DOMAIN.nsec:443 -showcerts`. Clean up to keep only the first certificate. Put it in `config.yaml` under `askgod_cert`.

## Usage

`./bin/linux/askgod-discourse config.yaml`
