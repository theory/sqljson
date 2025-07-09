#!/usr/bin/env perl -w

use strict;
use warnings;
use v5.30;

# Functions to cause SQL `true` and `false` args to jsonb_path_query() to evaluate.
sub true { 1 }
sub false { 0 }
sub NULL { undef }

my $num = 0;

while (<DATA>) {
    chomp;
    s/^select\s+(?:[*]\s+from\s+)?//i or next;
    my $comment = s/\s*--\s*(.+)// ? " // $1" : '';
    until (/;$/) {
        $_ .= <DATA>;
        chomp;
    }
    local $@;
    my ($json, $path, $opts) = eval;
    die $@ if $@;
    $num++;
    say qq/		{
			name: "test_$num",
			json: js(\`$json\`),
			path: \`$path\`,$opts
			exp:  []any{},$comment
		},/;
}

# Mock jsonb_path_query that converts its arguments into the JSON, path, and
# Options to specify the test.
sub jsonb_path_query {
    my ($json, $path, @opts) = @_;
    return $json, $path, '' unless @opts;
    my @options;
    while (@opts) {
        my $param = shift @opts;
        my $val = shift @opts;
        if ($param eq 'silent') {
            push @options => 'WithSilent()' if $val;
        } elsif ($param eq 'vars') {
            push @options => "WithVars(jv(\`$val\`))" if $val;
        } else {
            push @options => "WithVars(jv(\`$param\`))";
            push @options => 'WithSilent()' if $val;
        }
    }
    return $json, $path, "\n			opt:  []Option{" . join(',', @options) . '},';
}

sub jsonb_path_query_tz {
    my ($json, $path, $opts) = jsonb_path_query(@_);
    return $json, $path, "\n			opt:  []Option{WithTZ()}," unless $opts;
    $opts =~ s/\}$/WithTZ()}/;
    return $json, $path, $opts;
}

sub jsonb_path_query_array {
    jsonb_path_query(@_);
}

sub jsonb_path_query_first {
    jsonb_path_query(@_);
}

sub jsonb_path_match {
    jsonb_path_query(@_);
}

# Paste tests to convert below __DATA__.
__DATA__
SELECT jsonb_path_match('[{"a": 1}, {"a": 2}]', '$[*].a > 1');
SELECT jsonb_path_match('[{"a": 1}]', '$undefined_var');
SELECT jsonb_path_match('[{"a": 1}]', 'false');
