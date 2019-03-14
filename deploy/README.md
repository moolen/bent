# AWS ECS Mesh

This is a example deployment

## Installation

Make sure you have the [well-known AWS environment variables](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-envvars.html) set. This example uses [aws-vault](https://github.com/99designs/aws-vault) to load those variables.

Make sure you created a ssh keypair in aws beforehand.

```bash
$ ssh-add ~/Downloads/my-aws-keypair.pem
$ aws-vault exec ${MY_PROFILE} -- make plan keypair=my-aws-keypair
$ aws-vault exec ${MY_PROFILE} -- make apply keypair=my-aws-keypair
```

## What now?

```bash
$ ssh -L 16686:localhost:16686 ubuntu@$(terraform output jaeger)
```

Take a look at the traces and learn about how the services interact with each other.
