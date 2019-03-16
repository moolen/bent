resource "aws_ecs_service" "delta" {
  name            = "delta"
  cluster         = "${aws_ecs_cluster.white.id}"
  task_definition = "${aws_ecs_task_definition.delta.arn}"
  desired_count   = 2
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

data "template_file" "delta" {
  template = "${file("task-definitions/echo.json")}"
  vars {
      NAME = "delta.svc"
      ENVOY_XDS_HOST = "${aws_lb.xds.dns_name}"
      ENVOY_JAEGER_AGENT_HOST = "${aws_instance.jaeger.private_ip}"
  }
}

resource "aws_ecs_task_definition" "delta" {
  family                = "delta"
  container_definitions = "${data.template_file.delta.rendered}"
  execution_role_arn    = "${aws_iam_role.ecs_role.arn}"
  network_mode          = "awsvpc"

  requires_compatibilities = ["FARGATE"]
  cpu                      = "1024"
  memory                   = "2048"

  lifecycle {
    create_before_destroy = true
  }
}
