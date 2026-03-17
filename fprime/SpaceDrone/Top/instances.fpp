@ SpaceDrone Component Instances
@ Declares instances with base IDs, queue sizes, and priorities.

module SpaceDrone {

  module Default {
    constant QUEUE_SIZE = 10
    constant STACK_SIZE = 64 * 1024
  }

  # ------------------------------------------------------------------
  # Infrastructure instances
  # ------------------------------------------------------------------

  instance posixTime: Svc.PosixTime base id 0x0100

  instance rateGroupDriver: Svc.RateGroupDriver base id 0x0200

  instance rateGroup1: Svc.ActiveRateGroup base id 0x0300 \
    queue size Default.QUEUE_SIZE \
    stack size Default.STACK_SIZE \
    priority 55

  instance linuxTimer: Svc.LinuxTimer base id 0x0400

  # ------------------------------------------------------------------
  # SpaceDrone active component instances
  # ------------------------------------------------------------------

  instance missionControl: SpaceDrone.MissionControl base id 0x1000 \
    queue size Default.QUEUE_SIZE \
    stack size Default.STACK_SIZE \
    priority 50

  instance gnc: SpaceDrone.GNC base id 0x4000 \
    queue size Default.QUEUE_SIZE \
    stack size Default.STACK_SIZE \
    priority 45

  instance powerManagement: SpaceDrone.PowerManagement base id 0x5000 \
    queue size Default.QUEUE_SIZE \
    stack size Default.STACK_SIZE \
    priority 40

  instance communications: SpaceDrone.Communications base id 0x6000 \
    queue size Default.QUEUE_SIZE \
    stack size Default.STACK_SIZE \
    priority 35

  instance objectDetection: SpaceDrone.ObjectDetection base id 0x7000 \
    queue size Default.QUEUE_SIZE \
    stack size Default.STACK_SIZE \
    priority 48

  instance healthMonitor: SpaceDrone.HealthMonitor base id 0x8000 \
    queue size Default.QUEUE_SIZE \
    stack size Default.STACK_SIZE \
    priority 30

}
