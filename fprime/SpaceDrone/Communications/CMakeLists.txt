####
# Communications Component
# CCSDS Space Packet Protocol (133.0-B-2)
# S-band ground link + UHF backup + SpaceWire mesh
####
set(SOURCE_FILES
    "${CMAKE_CURRENT_LIST_DIR}/Communications.fpp"
)
set(MOD_DEPS
    SpaceDrone_Types
)
register_fprime_module()
