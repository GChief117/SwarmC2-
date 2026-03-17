####
# GNC Component — Guidance, Navigation & Control
# 400Hz attitude control, 100Hz guidance, 10Hz navigation
####
set(SOURCE_FILES
    "${CMAKE_CURRENT_LIST_DIR}/GNC.fpp"
)
set(MOD_DEPS
    SpaceDrone_Types
)
register_fprime_module()
