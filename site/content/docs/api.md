# JSON API Monitor

The JSON API monitor is designed to deeply validate responses from JSON endpoints. Rather than just checking if an endpoint returns a `200 OK`, it allows you to verify that specific keys exist and contain expected values.

## Configuration Options

When setting up an API monitor, you can configure the following options:

### Basic Settings

- **Name:** A descriptive name for your monitor.
- **Group:** Optional group assignment for organizing monitors.
- **Interval (seconds):** How frequently updu should check the API.
- **Timeout (seconds):** The maximum time updu will wait for a response.

### API Specific Settings

- **URL:** The full API endpoint to query (e.g., `https://api.example.com/v1/status`).
- **Method:** The HTTP method to use (GET, POST, PUT, etc.).
- **Headers:** (Optional) Custom HTTP headers to send with the request, such as `Authorization: Bearer <token>`.
- **Body:** (Optional) The JSON payload to send if using methods like POST or PUT.
- **Expected Status:** The HTTP status code expected (e.g., `200`).
- **JSON Validation Rules:** This is the core feature of the API monitor. You can define multiple rules to evaluate the response body.
  - **JSON Path:** use dot notation (e.g., `data.status`) or bracket notation to target specific fields in the JSON response payload.
  - **Expected Value:** The value that the field targeted by the JSON Path *must* equal for the check to pass.

## Example Use Cases

- **Advanced Health Checks:** Query a `/health` endpoint and verify that `{"database_status": "connected"}` is present in the response.
- **Critical Integrations:** Ensure a third-party API is returning the expected data structure before your application attempts to parse it.
