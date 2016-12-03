Withenv
-------

Withenv takes YAML files and applies them to the environment prior to
running a command. The idea is that rather than relying on shell
variables being set, you can explicitly define what environment
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

See the [Python Docs](https://withenv.readthedocs.org) for info.
