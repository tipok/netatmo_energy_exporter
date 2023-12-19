# Netatmo Energy Exporter

This Prometheus exporter works with the netatmo energy API.
It reads the current temperature measurement and set point temperature
and exports it in prometheus readable way alongside with other metrics.
This exporter publishes metrics per room and per modules.

*IMPORTANT*: this exporter works only with netatmo Thermostats and Valves.

## Build Docker Image

The best way to deploy is by creating a docker image by executing:

```shell
docker build -t netatmo_energy_exporter .
```

## Run Docker Container

1. First of all create an App in netatmos developers portal
2. Generate and copy the client id and secret
   * if you're going to use the refresh token, generate one and copy it
3. Run by executing:
    ```shell script
    docker run -d -p 2112:2112 netatmo_energy_exporter \
       --client-id=${CLIENT_ID} --client-secret=${CLIENT_SECRET} \
       --username=${USERNAME} --password=${PASSWORD}
    ```
   or
   ```shell script
   docker run -d -p 2112:2112 netatmo_energy_exporter \
      --client-id=${CLIENT_ID} --client-secret=${CLIENT_SECRET} \
      --refresh-token=${REFRESH_TOKEN}
   ```
   
### Using refresh token

Netatmo has deprecated the ability to use the password credential flow, even though it's still listed.
If you're experiencing any issues while using your username + password combination, try to get the 
refresh token. You should use the following scopes while generating the token:
- read_station
- read_thermostat

### Supported CLI Arguments

--client-id :: netatmo APP client id [*required*]

--client-secret :: netatmo APP client secret [*required*]

--username :: netatmo username [*required*]

--password :: netatmo password [*required*]

--refresh-token :: netatmo refresh token [*required*]

--listen :: address in default go format to listen to (default _0.0.0.0:2112_) [*optional*]
