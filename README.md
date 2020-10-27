cf-protect - A Cloud Foundry Plugin for SHIELD Cloud
====================================================

There's important data in the services _behind_ your Cloud Foundry
applications, data you care about that is either impossible or at
least costly to recreate should it go missing.

That's what `cf-protect` is designed to do!  This small CF CLI
plugin analyzes applications for bound data services and
automatically configures your SHIELD Cloud account to perform
regularly scheduled backups of those data systems.

Here's an example:

```shell
$ cf apps
name   requested state   instances   memory   disk   urls
todo   started           1/1         64M      1G     todo.cfapps.io

$ cf services
name      service   plan    bound apps   last operation     broker      upgrade available
todo-db   cleardb   spark   todo         create succeeded   appdirect

$ cf protect todo --shield-agent agent@demo
protecting application todo
Connecting to SHIELD...

protecting service todo-db (mysql):
  hostname: us-cdbr-east-02.cleardb.com
  port:     3306
  database: ad_29e123458039d1e
  username: **************
  password: ********

created system starkandwayne/demos/todo/todo-db [3b0d69de-3378-4eeb-a9f4-c383ccdbed15]...
created job Daily [86677ee8-e63f-4eb4-baf6-c5130c9404a1]...
```

## Installation

The easiest way to install this plugin for your CF CLI is to use
our public repository:

```shell
$ cf add-plugin-repo starkandwayne https://cf.pub.starkandwayne.com/
$ cf install-plugin -r starkandwayne cf-protect
```

If you'd prefer to compile it form source, you are welcome to do
that as well.  Just clone this git repository from GitHub
(<https://github.com/shieldcloud/cf-protect>), and run `make
install`:

```shell
$ git clone https://github.com/shieldcloud/cf-protect
$ cd cf-protect

$ make install
cf uninstall-plugin protect || true
Plugin protect does not exist.
FAILED
yes | cf install-plugin cf-protect
Attention: Plugins are binaries written by potentially untrusted authors.
Install and use plugins at your own risk.
Do you want to install the plugin cf-protect? [yN]: y
Installing plugin protect...
OK

Plugin protect N/A successfully installed.
```

_Note:_ the `make install` target tries to uninstall the plugin,
which will always fail if this is your first time installing.
That's okay!

## Usage

Before you can use this plugin, you will need to sign up for a
[SHIELD Cloud account][1].  Once your SHIELD is spun up, you've
targeted it with the `shield` CLI, and have installed at least one
agent (all of which is covered in the shieldcloud.io docs), you
can point `cf-protect` at your CF applications like this:

```shell
$ cf-protect APP-NAME --core NAME --agent NAME
```

## What Data Services Are Supported?

Currently, we support MySQL and PostgreSQL, doing single-database
backups.  The service instances will only be recognized if they
are tagged as `mysql` and `postgresql` (respectively), and have
the appropriate instance binding credentials.

For MySQL service instances, those credentials are:

  1. `hostname` - the hostname or IP address of the MySQL server.
  2. `port` - The port where MySQL is listening.
  3. `name` - The name of the database.
  4. `username` - Username for authentication purposes.
  5. `password` - Secret password for authentication purposes.

For PostgreSQL service instances, those credentials are:

  1. `uri` - a PostgreSQL URI containing the hostname and port,
     username and password, and database name.

We are planning on expanding these heuristics to include more data
services as we encounter them in the wild and as people request
them.  If you have a need that isn't currently served, please open
a [GitHub Issue][2] and we'll try to get it supported!


[1]: https://shieldcloud.io/
[2]: https://github.com/shieldcloud/cf-protect/issues
