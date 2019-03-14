resource "aws_ecs_service" "xds" {
  name            = "xds"
  cluster         = "${aws_ecs_cluster.red.id}"
  task_definition = "${aws_ecs_task_definition.xds.arn}"
  desired_count   = 1
  launch_type     = "FARGATE"

  load_balancer {
    target_group_arn = "${aws_lb_target_group.xds.arn}"
    container_name   = "bent"
    container_port   = 50000
  }

  network_configuration = {
    subnets = [
      "${aws_subnet.private-a.id}",
      "${aws_subnet.private-b.id}",
    ]

    security_groups = [
      "${aws_security_group.all.id}",
    ]
  }
}

data "template_file" "xds" {
  template = "${file("task-definitions/xds.json")}"
}

resource "aws_ecs_task_definition" "xds" {
  family                = "xds"
  container_definitions = "${data.template_file.xds.rendered}"
  network_mode          = "awsvpc"
  execution_role_arn    = "${aws_iam_role.ecs_role.arn}"
  task_role_arn          = "${aws_iam_role.ecs_role.arn}"

  requires_compatibilities = ["FARGATE"]
  cpu                      = "1024"
  memory                   = "2048"

  lifecycle {
    create_before_destroy = true
  }
}

# we need a loadbalancer
resource "aws_lb" "xds" {
  name                       = "xds-lb"
  internal                   = true
  load_balancer_type         = "network"
  enable_deletion_protection = false

  subnets = [
    "${aws_subnet.private-a.id}",
    "${aws_subnet.private-b.id}",
  ]
}

resource "aws_lb_target_group" "xds" {
  name        = "xds-lb-tg"
  target_type = "ip"
  port        = 50000
  protocol    = "TCP"
  vpc_id      = "${aws_vpc.main.id}"

  health_check {
    port     = "50000"
    protocol = "TCP"
    interval = "30"
  }
}

resource "aws_lb_listener" "xds" {
  load_balancer_arn = "${aws_lb.xds.arn}"
  port              = "50000"
  protocol          = "TCP"

  default_action {
    type             = "forward"
    target_group_arn = "${aws_lb_target_group.xds.arn}"
  }

  depends_on = [
    "aws_lb.xds",
  ]
}
