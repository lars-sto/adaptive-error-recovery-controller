# Adaptive Error Recovery Controller

A modular recovery policy system for adaptive protection

The current implementation focuses on FlexFEC, but the code is already structured into layered components 
so that additional mechanisms and models can be integrated later.

## Current Architecture

The current codebase implements the following layers:

1. Observation Layer
    - Not yet implemented
    - wiring of runtime signal producers such as stats interceptors, transport feedback adapters, or GCC inputs

2. Coordination Layer
    - Not yet implemented
    - event-driven coordination and shared runtime state aggregation
    - planned for later integration work
    - Orchestrator Component that collects Input from BWE, Stats, ... and produces snapshots for Orchestrator

3. Orchestration Layer
    - orchestrates runtime decision flow
    - consumes runtime stats
    - invokes mechanism-specific controllers
    - publishes policy decisions

4. Mechanism Layer
    1. Model
       - defines mechanism specific decision logic
    2. Controller
       - transforms model into runtime policy
       - does claming, hysteresis, stability logic, ...
       - decides if and how policy updates should be propagated
    3. Interceptors
       - applies policies
       - execution layer
    - currently implemented: FlexFEC
    - future extension points: NACK, RED, audio FEC, ...

## Current Scope

The current implementation is limited to:
- one concrete mechanism implemented: FlexFEC-03
- one recovery engine instance
- supports multiple mechanism controllers per sample

