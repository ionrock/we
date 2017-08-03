========================
 Examples Using Withenv
========================

Withenv can be helpful organizing and using environment variables more
consistently, but it can also be used as the basis for complex
workflows that make transitioning from development to production
consistent and predictable.


Configuration Management
========================

One example where withenv can be valuable is in the context of
configuration management. Most configuration management systems push
(or pull) changes to a system explicitly or based on a cadence. The
result is that you often have to execute the configuration management
tooling on a system in order to get changes for a single
application. What is more, these "deploy time" changes don't allow a
great deal of dynamism where it is valueable.

Withenv can be a valuable tool to help reduce dependency on
configuration management systems in order to get dynamic "runtime"
values along side simplifying configuration management data
management. Lets take a look at how this works.

Lets assume that we are deploying a chat bot called `webot` to some
cloud VMs. In our repository we create an `envs` directory to keep
configuration YAML that we'll use with `we` for defining how to run
`webot` in development, staging, CI and production. Our bot is going
to connect to Slack so there are some secret keys we want to use as
well as a database we'll connect to depending on the context. When we
deploy the app to the VM, we'll package things up in a docker
container.

So our `webot` repo looks something like this:

.. code-block::

   webot/
     Makefile
     webot.go
     envs/
       dev/
       stage/
       prod/

Our `Makefile` has some targets to run our bot for development. Each
target can use a `ENV` flag to use different configuration. The
targets look something like this:

.. code-block:: bash

   ENV=dev

   run:
   	we -a envs/$(ENV)/main.yml webot

We'd run `make run` to run with development settings and `make -e
ENV=stage run` to use staging settings. Each settings can include the
necessary configuration such as Slack and database config. In
development, that might be a personal development key encrypted
using gpg locally or a predefined key loaded from a secret store. The
database configuration might use an in memory database for development
and a MySQL instance in staging and production.

In order to build new docker containers, we'll copy in the config we
want to use as well as include `we` so we can run the command the same
way we do locally.

.. code-block::

   FROM ubuntu

   ENV ENV=stage

   RUN mkdir -p /opt/webot/envs/$ENV
   COPY envs/$ENV/:/opt/webot/envs/$ENV

   CMD we -a envs/$ENV/main.yml webot

We can test what our config looks like in production by running
`we --clean -a envs/$ENV/main.yml` to see what is getting set in the
environment and what values they are set to. This can happen locally
without having to use something like `test-kitchen
<http://kitchen.ci/>`_ to exercise the configuration management
code.

So, far we see how to:

 - keep our config next to the code
 - keep secrets stored safely based on context
 - package code with configuration

Finally, we'd like to think about how to add more dynamic elements to
our config. For example, lets use a service discovery tool like
`Consul <https://consul.io>`_ to find our database. In our production
config we can then change our database URI from an explicit value to
something that uses Consul.

.. code-block:: yaml

   ---
   DB_URI: `consul kv get webot/$ENV/database | jq .DB_URI`

Now, when the process starts or restarts, it will query service
discovery to find the necessary `DB_URI` value. We can use the same
methodology for secrets where we could use vault to get the value from
the secret store at runtime.

Finally, lets consider how this change effects the configuration
management. We can likely remove explicit references to services such
as the database and any secrets. If there was code to find the
database (using chef search or dynamic inventory in ansible), this can
also be removed. Similarly, any other data that is necessary for
configuration for `webot` can be removed. While this doesn't seem
like a huge change, it allows the configuration to be tested by
developers in the standard CI pipeline. Developers do not have to
learn the configuration management tooling or its environment to
reliably update configuration.

Outside of our `webot` app, there are likely other services running on
our VM that use configuration management. It is still reasonable to
consider using `we` for these processes as it helps to decouple your
configuration data with your configuration management tooling. Often
times it would be beneficial to use more than one configuration
management tool based on its strengths. For example, if you maintain
a basic system with Chef, but provision specific applications with
Ansible. By using `we`, you can define your configuration data as YAML
that can be used in any configuration management system.


Upstream Dependencies
=====================

If you don't use a configuration management system and instead use
something like `Kubernetes <https://kubernetes.io/>`_, `we` can be
helpful limiting upstream dependencies. As we saw in the configuration
management example, we were able to package our configuration with a
docker container and then use `we` to inject that configuration at
runtime. Kubernetes provides a means of injecting your environment
directly into your container, but this creates an implicit upstream
dependency.

Lets consider a container that we built that is dependent on the
upstream system providing environment variables for
configuration. First step before this container reaches production
should be to run some tests in a CI system. The CI system then needs
to fulfill the same contract as the production system. There might
need to be some code needed to make this happen depending on the CI
system. Also, if the production system will inject secrets, you'll
need to figure out a similar mechanism in your CI system as
well. Finally, if there are any changes in the upstream production
system in how it injects configuration (including name changes), that
change needs to be reimplemented in your CI system. The same principle
is in place for any sort of staging system or tooling that provide
ephemeral environments.

By using `we` you can abstract the data and injection consistently
between all the systems. This abstraction works just as well on a
laptop, VM or in a container. You can still use upstream tooling, but
the use can be explicit.

Lets consider an example where a Kubernetes cluster will inject a TLS
cert into your container for ensuring communication is secure between
services. Your application accepts a `CLIENT_CERT` environment
variable to use when connecting to services. The Kubernetes
environment provides a `MYORG_CRT` variable to use for the cert
value. You production config can include the mapping.

.. code-block:: yaml

   ---
   - CLIENT_CERT: $MYORG_CRT

The mapping can change based on the context such as in CI or staging
where the cert might be limited to non-production resources.

There is also the consideration of changing things over time. Lets
assume that rather than a cert, a private key has been made
available. When the mechanism was first created, the key was a 256 bit
RSA key, but the organization needs to upgrade to a 1024 bit key. The
upstream system might begin inserting both keys during the
transition using a `MYORG_KEY_1024` along side a `MYORG_KEY`. Each
project can change the mapping using the `we` YAML files. Those same
files can be used to track the migration and provide a means of
auditing the process. For example, if each team used a `envs/$ENV`
pattern, it would be trivial to write a shell script to find where
`prod` mappings are still using the old `MYORG_KEY`.



Conclusions
===========

It can be subtle how `we` can improve your development and operational
environment. Withenv provides a valuable abstraction for explicitly
defining the data an application needs.
