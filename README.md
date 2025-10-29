# Fault-Tolerant-Microservice

## Overview
Demonstrated fault-tolerant system design by implementing circuit breaker and fail-fast patterns 
to handle service failures and cascading errors in a distributed application built with Go.

## Problem Statement
- Built a microservice that intentionally crashes under load
- Demonstrated cascading failures without proper resilience patterns
- Showed system degradation through load testing metrics

## Solution Approach
- Implemented Circuit Breaker pattern using [library/custom implementation]
- Added fail-fast mechanisms to prevent cascade failures
- [If applicable: Added bulkhead pattern for resource isolation]
