1. naming conventions

type: PascalCase
method: PascalCase
member variable: snake_case
interface: I + PascalCase
local variable: snake_case
function: camelCase


2. file name conventions

interface: name
implementation(when interface file is seperated): name + _impl
implementation(without seperated interface file): name


3. work process

1) write interface file
2) write test file
3) make it compileable
4) implement


4. clone/copy

1) all interface must be copyable, and it must not copy the implementation.
2) always copy pass interface
3) all interface is the pointer of a struct.
4) never define methods for non-interface-implementing structs -> choose between [method-only interface(+implementation struct) | variable-only struct]

* interface-implementing structs for polymorphism of simple types are exceptions.
* never use polymorphism for large types.

5. fail-safe programming

1) golang program can crash at any point
2) functions doing a. communication b. disk access c. memory allocation d. calling other failable function
must return (result, error)
3) always check if err != nil
4) never let a handled error crashs the whole system.

5. Construction

1) All interface implementing structs must be constructed with New___() functions.
These functions return the struct pointer, not interface.

2) Optionally, struct may have Init() method, which is expected to be called by composition struct construction, only once.