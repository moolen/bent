data "aws_ami" "ubuntu" {
  most_recent = true

  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd/ubuntu-bionic-18.04-amd64-server-*"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }

  owners = ["099720109477"] # Canonical
}

data "template_file" "jaeger_provision" {
  template = "${file("scripts/jaeger.provision.sh")}"
}

resource "aws_instance" "jaeger" {
  ami           = "${data.aws_ami.ubuntu.id}"
  instance_type = "t2.micro"
  key_name      = "${var.keypair}"

  subnet_id = "${aws_subnet.public.id}"

  vpc_security_group_ids = [
    "${aws_security_group.public_ssh.id}",
    "${aws_security_group.all.id}",
  ]

  associate_public_ip_address = true

  user_data = "${data.template_file.jaeger_provision.rendered}"

  lifecycle {
    create_before_destroy = true
  }
}


output "jaeger" {
  value = "${aws_instance.jaeger.public_ip}"
}
