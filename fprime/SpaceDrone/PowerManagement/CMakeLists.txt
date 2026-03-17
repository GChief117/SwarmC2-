####
# PowerManagement Component
# Theorem 1: Energy Invariance — E(C') ≤ B ≤ (Ps+Pb)·T
####
set(SOURCE_FILES
    "${CMAKE_CURRENT_LIST_DIR}/PowerManagement.fpp"
)
set(MOD_DEPS
    SpaceDrone_Types
)
register_fprime_module()
