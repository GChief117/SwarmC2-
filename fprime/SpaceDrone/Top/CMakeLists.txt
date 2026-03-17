####
# SpaceDrone Topology — Top-level component wiring
####
set(SOURCE_FILES
    "${CMAKE_CURRENT_LIST_DIR}/instances.fpp"
    "${CMAKE_CURRENT_LIST_DIR}/SpaceDroneTopology.fpp"
)

set(MOD_DEPS
    SpaceDrone_Types
    SpaceDrone_MissionControl
    SpaceDrone_GNC
    SpaceDrone_PowerManagement
    SpaceDrone_Communications
    SpaceDrone_ObjectDetection
    SpaceDrone_HealthMonitor
)

register_fprime_module()
