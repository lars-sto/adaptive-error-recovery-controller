# Adaptive Error Recovery Controller

A modular policy engine for adaptive Forward Error Correction (FEC) in real-time media systems.

This project implements a cleanly separated policy layer that decides when and how much FEC protection should be applied based on observed network conditions (loss, RTT, bitrate constraints).

It is designed to integrate with WebRTC systems (e.g., Pion) but remains transport-agnostic.