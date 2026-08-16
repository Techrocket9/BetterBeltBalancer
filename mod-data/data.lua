-- The data stage, which is Lua and stays Lua.
--
-- Factorio's data stage is a declarative prototype dump with no runtime and no
-- state: there is nothing here for a Go guest to be the brain of, and
-- `fklua mod` generates only the control stage. So these files are hand-written
-- and OVERLAID onto the packaged mod by `make mod` -- see CLAUDE.md.
require("prototypes.entity")
require("prototypes.hidden")
require("prototypes.sprite")
require("prototypes.item")
require("prototypes.recipe")
require("prototypes.technology")
