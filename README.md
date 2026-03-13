# Adaptive Error Recovery Controller

A modular recovery policy system for adaptive protection

The current implementation focuses on FlexFEC, but the code is already structured into layered components 
so that additional mechanisms and models can be integrated later.

## Current Architecture

The prototype is organized into four layers:

1. Observation Layer
    - planned input layer for runtime signals
    - examples: stream statistics, feedback adapters, bandwidth estimation inputs

2. Coordination Layer
    - planned shared-state and event-coordination layer
    - event-driven coordination and shared runtime state aggregation
    - intended to combine inputs from multiple signal sources into coherent runtime snapshots

3. Orchestration Layer
    - drives runtime decision flow
    - consumes runtime observations
    - invokes mechanism-specific controllers
    - publishes policy decisions

4. Mechanism Layer
    - contains mechanism-specific adaptation logic
    - currently implemented for FlexFEC
    - split into:
        - Model: decision logic
        - Controller: policy shaping, hysteresis, stability logic, bounded updates
        - Interceptor integration: applies policy decisions at runtime

Status

This repository is currently a prototype used to explore adaptive FlexFEC control and possible integration patterns for runtime coordination between interceptors and related components.

Possible future extensions include NACK, RED, audio FEC, and additional signal sources.
