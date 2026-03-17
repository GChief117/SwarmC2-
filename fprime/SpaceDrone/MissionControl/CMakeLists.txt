####
# MissionControl Component
# Central FSM controller: BOOT/CRUISE/EVADE/HOLD/SAFE_MODE
# Theorem 2: FSM Completeness — 5 states × 4 inputs = 20 defined pairs
####
set(SOURCE_FILES
    "${CMAKE_CURRENT_LIST_DIR}/MissionControl.fpp"
)
set(MOD_DEPS
    SpaceDrone_Types
)
register_fprime_module()
