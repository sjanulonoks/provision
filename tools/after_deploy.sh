#!/bin/bash

curl -H "Content-Type: application/json" --data "{\"source_type\": \"Tag\", \"source_name\": \"$TRAVIS_TAG\"}" -X POST https://registry.hub.docker.com/u/digitalrebar/provision/trigger/f3e8b588-59d7-43f7-b855-b49019de2dee/
