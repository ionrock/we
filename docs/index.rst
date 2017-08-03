Welcome to withenv's documentation!
======================================

Withenv (`we`) is a tool to help manage injecting configuration data
into applications. Applications typically need some information in
order to run properly.

The idea of withenv started as an easy way set environment variables
prior to running some program that depends on them, leaving your shell
in a sane state.

Installation
------------

Grab the [latest we release](https://github.com/ionrock/we/releases)
and add it to your path.


Quick Start
-----------

Withenv takes YAML/JSON files and applies them to the environment
prior to running a command. The idea is that rather than relying on
shell variables being set, you can explicitly define what environment
variables you want to use.

For example, here is some YAML that might be helpful when connecting
to an OpenStack Cloud.

```yaml
---
OS_USERNAME: user
OS_PASSWORD: secret
OS_URL: http://api.example.org/rest/api
```

You can then run commands (ie in Ansible for example) that read this
information.

```bash
$ we -e test_user_creds.yml ansible-play install-app
```


Contents:

.. toctree::
   :maxdepth: 2

   usage
   contributing
   authors

Indices and tables
==================

* :ref:`genindex`
* :ref:`modindex`
* :ref:`search`
