# Adaptive Error Recovery Controller

A modular recovery policy system for adaptive protection

The current implementation focuses on FlexFEC, but the code is already structured into layered components 
so that additional mechanisms and models can be integrated later.

## Current Architecture

The current codebase implements the following layers:

1. **Observation Layer**
    - Not yet implemented
    - wiring of runtime signal producers such as stats interceptors, transport feedback adapters, or GCC inputs

2. **Coordination Layer**
    - Not yet implemented
    - event-driven coordination and shared runtime state aggregation
    - planned for later integration work

3. **Recovery Layer**
    - orchestrates runtime decision flow
    - consumes runtime stats
    - invokes mechanism-specific controllers
    - publishes policy decisions

4. **Mechanism Layer**
    - contains mechanism-specific recovery controllers
    - currently implemented: **FlexFEC**
    - future extension points: NACK, RED, audio FEC, ...

5. **Model Layer**
    - contains pluggable decision logic used by a mechanism controller
    - currently implemented: a table-based FlexFEC model

## Current Scope

The current implementation is intentionally limited to:
- one concrete mechanism implemented: **FlexFEC-03**
- one recovery engine instance
- supports multiple mechanism controllers per sample

## Adding a New Mechanism

To add a new recovery mechanism such as RED, the following steps are typically required:

1. **Define the mechanism kind**
    - add a new `MechanismKind` constant

2. **Define the mechanism-specific policy type**
    - for example `REDPolicy`

3. **Implement a mechanism-specific controller**
    - implement the `MechanismController` interface
    - handle mechanism-specific state transitions and policy construction

4. **Add an optional model layer**
    - define a model interface if the mechanism should support pluggable decision logic
    - for example a rule-based or table-based RED model

5. **Register the controller**
    - extend `controller_factory.go` so the recovery engine can instantiate the mechanism

6. **Add policy extraction helpers**
    - add helper functions similar to `AsFlexFECPolicy(...)`

7. **Add tests**
    - controller tests
    - model tests if applicable
    - integration tests at the recovery engine level if needed

### Example: adding RED support

Adding RED would likely involve:
- introducing `MechanismRED`
- defining a `REDPolicy`
- implementing a `REDController`
- optionally adding a `REDModel`
- extending the controller factory
- adding a helper such as `AsREDPolicy(...)`

## Adding a New Model

A mechanism can support multiple interchangeable decision models.

### Example: adding a new FlexFEC model

The current FlexFEC controller uses the `FlexFECModel` interface. To add a new model, the following steps are typically required:

1. **Implement the model interface**
    - implement `FlexFECModel`
    - provide a `Recommend(NetworkStats) FlexFECRecommendation` method

2. **Return a raw recommendation**
    - the model should return a `FlexFECRecommendation`
    - for example a target overhead and an optional reason string

3. **Keep enforcement inside the controller**
    - hysteresis
    - deadband handling
    - bandwidth caps
    - actuator mapping such as `(k, r)`
    - these remain in `FlexFECController`

4. **Inject the model into the controller**
    - pass the model into `NewFlexFECController(...)`
    - if no model is provided, the default table-based model is used

### Current example

The current default implementation is:
- `FlexFECModel`
- `TableBasedFlexFECModel`

