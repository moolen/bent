resource "aws_ecs_service" "gamma" {
  name            = "gamma"
  cluster         = "${aws_ecs_cluster.red.id}"
  task_definition = "${aws_ecs_task_definition.gamma.arn}"
  desired_count   = 1
  launch_type     = "FARGATE"

  network_configuration = {
    subnets = [
      "${aws_subnet.private-a.id}",
    ]

    security_groups = [
      "${aws_security_group.all.id}",
    ]
  }

}

data "template_file" "gamma" {
  template = "${file("task-definitions/fwd.json")}"
  vars {
      NAME = "gamma.svc"
      TARGET = "http://delta.svc/from/gamma,http://eta.svc/from/gamma,http://epsilon.svc/from/gamma"
      ENVOY_XDS_HOST = "${aws_lb.xds.dns_name}"
      ENVOY_JAEGER_AGENT_HOST = "${aws_instance.jaeger.private_ip}"
  }
}

resource "aws_ecs_task_definition" "gamma" {
  family                = "gamma"
  container_definitions = "${data.template_file.gamma.rendered}"
  execution_role_arn    = "${aws_iam_role.ecs_role.arn}"
  network_mode          = "awsvpc"

  requires_compatibilities = ["FARGATE"]
  cpu                      = "1024"
  memory                   = "2048"

  lifecycle {
    create_before_destroy = true
  }
}
