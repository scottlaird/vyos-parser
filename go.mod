module github.com/scottlaird/vyos-parser

go 1.22.2

replace github.com/scottlaird/vyos-parser/configmodel => ./configmodel
replace github.com/scottlaird/vyos-parser/syntax => ./syntax

require (
        github.com/hexops/gotextdiff v1.0.3
        github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
        github.com/scottlaird/vyos-parser/configmodel v0.0.0-20250119183125-82a1aa2032f5
)
