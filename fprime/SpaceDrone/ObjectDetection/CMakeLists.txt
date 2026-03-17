####
# ObjectDetection Component
# LiDAR-based proximity detection and collision avoidance
####
set(SOURCE_FILES
    "${CMAKE_CURRENT_LIST_DIR}/ObjectDetection.fpp"
)
set(MOD_DEPS
    SpaceDrone_Types
)
register_fprime_module()
