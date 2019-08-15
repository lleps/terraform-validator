This tool is used to manipulate the [terraform-compliance](https://github.com/eerkunt/terraform-compliance/) 
tool and its state through a REST API, allowing devs to test terraform 
plan files against specific security requirements.

It contains a client to execute the requests, and a server which wraps
the tool and implements the API.

# API specification

### `POST /validate`
Validate a terraform plan file with the current features. The plan file
content must be passed as a base64 string in the body.

### `GET /features`
List all the used features.

### `GET /features/source/{name}`
Get the source code of the `name` feature.

### `POST /features/add/{name}`
Add a new feature with `name`. The feature source code is also passed
in the request body, as a raw string.

The syntax used to describe features is specified [here](https://github.com/eerkunt/terraform-compliance/blob/master/README.md).

New calls to /validate will use this new feature to test the plan file.

### `DELETE /features/delete/{name}`
Delete the `name` feature, so it's not used to test plan
files anymore.