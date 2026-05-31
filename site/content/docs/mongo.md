# MongoDB Monitor

The MongoDB monitor in updu provides direct checking against MongoDB endpoints or clusters. It connects natively to the requested document database layer and issues a `ping` command on the `admin` schema.

## Configuration Options

When setting up a Mongo monitor, you can configure the following options:

### Basic Settings

- **Name:** A descriptive name for your monitor.
- **Group:** Optional group assignment for organizing monitors.
- **Interval (seconds):** How frequently updu should perform the check.
- **Timeout (seconds):** The maximum time updu will wait for a response before considering the check failed.

### MongoDB Specific Settings

- **Connection URI:** The full connection URL to a cluster or server exposing the `mongod` instance. A standard prefix looks like `mongodb://username:password@localhost:27017`, while clustered setups rely on a slightly different structure such as `mongodb+srv://admin:pass@cluster.mongodb.net/`. updu extracts the parameters needed to verify the database and authenticate automatically from these URI patterns.

## Example Use Cases

- **Mongo Deployments:** Ensure your MongoDB replica set is online and the primary is available to accept data operations.
