####
# HealthMonitor Component
# Subsystem watchdog timer (Section 6.3)
####
set(SOURCE_FILES
    "${CMAKE_CURRENT_LIST_DIR}/HealthMonitor.fpp"
)
set(MOD_DEPS
    SpaceDrone_Types
)
register_fprime_module()
